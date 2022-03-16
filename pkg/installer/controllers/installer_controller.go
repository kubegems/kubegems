/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type InstallerOptions struct {
	ChartsDir string `json:"chartsDir,omitempty"`
}

// InstallerReconciler reconciles a Memcached object
type InstallerReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Options    *InstallerOptions
	HelmConfig *action.Configuration
}

//+kubebuilder:rbac:groups=kubegems.io,resources=installers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubegems.io,resources=installers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubegems.io,resources=installers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *InstallerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	installer := &pluginsv1beta1.Installer{}
	if err := r.Client.Get(ctx, req.NamespacedName, installer); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.Sync(ctx, installer); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *InstallerReconciler) Sync(ctx context.Context, installer *pluginsv1beta1.Installer) error {
	// load helm
	log := logr.FromContext(ctx)
	log.Info("reconciling")
	defer log.Info("reconciled")

	// one by one in dep order
	for _, plugin := range installer.Spec.Plugins {
		if plugin.Name == "" || plugin.Namespace == "" {
			continue
		}
		// extract current status
		status := extractCurrentStatus(plugin, installer)
		originalStatus := status.DeepCopy()
		if err := r.sync(ctx, plugin, status); err != nil {
			resetStatus(status)
			status.Status = pluginsv1beta1.StatusFailed
			status.Message = err.Error()
			if err := r.Status().Update(ctx, installer); err != nil {
				return err
			}
			return err
		}
		// not status update,continue next plugin
		// https://github.com/golang/go/issues/19502
		if apiequality.Semantic.DeepEqual(originalStatus, status) {
			continue
		}
		if err := r.Status().Update(ctx, installer); err != nil {
			return err
		}
	}
	return nil
}

// nolint: funlen,nestif
func (r *InstallerReconciler) sync(ctx context.Context, plugin pluginsv1beta1.InstallerSpecPlugin, status *pluginsv1beta1.InstallerStatusStatus) error {
	namespace, name := plugin.Namespace, plugin.Name
	log := logr.FromContext(ctx).WithValues("name", name, "namespace", namespace)

	config := NewHelmConfig(namespace)

	// do get first
	isNotFound := false
	exist, err := action.NewGet(config).Run(name)
	if err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			return err
		}
		isNotFound = true
	}

	if !plugin.Enabled {
		if isNotFound {
			if status.Status == pluginsv1beta1.StatusUninstalled || status.Status == pluginsv1beta1.StatusNotInstall {
				return nil
			}
			log.Info("plugin is disabled and not installed")
			status.Status = pluginsv1beta1.StatusNotInstall
			return nil
		}
		// record exist and uninstalled
		if exist.Info.Status == release.StatusUninstalled {
			log.Info("plugin is disabled and uninstalled")
			updateStatusFromRelease(status, exist)
			return nil
		}
		// uninstall
		log.Info("uninstalling")
		uninstall := action.NewUninstall(config)
		uninstalledRelease, err := uninstall.Run(name)
		if err != nil {
			log.Error(err, "on uninstall")
			return err
		}
		log.Info("uninstalled")
		if urelease := uninstalledRelease.Release; urelease != nil {
			updateStatusFromRelease(status, urelease)
		} else {
			log.Info("uninstalled, but no release")
			status.Message = uninstalledRelease.Info
			status.Status = pluginsv1beta1.StatusUninstalled
		}
		return nil
	}

	values := unmarshalValues(plugin.Values) // ignore error
	loader, err := loader.Loader(filepath.Join(r.Options.ChartsDir, name))
	if err != nil {
		log.Error(err, "on chart loader")
		return err
	}
	chart, err := loader.Load()
	if err != nil {
		log.Error(err, "on chart load")
		return err
	}

	if isNotFound {
		// install
		log.Info("installing")
		install := action.NewInstall(config)
		install.ReleaseName = name
		install.CreateNamespace = true
		install.Namespace = namespace
		installedRelease, err := install.Run(chart, values)
		if err != nil {
			log.Info("on install")
			return err
		}
		log.Info("installed")
		updateStatusFromRelease(status, installedRelease)
		return nil
	}

	// should upgrade
	if exist.Info.Status == release.StatusDeployed && reflect.DeepEqual(exist.Config, values) {
		log.V(5).Info("already uptodate")
		updateStatusFromRelease(status, exist)
		return nil
	}

	// upgrade
	log.Info("upgrading")
	log.V(5).Info("status not healthy", "status", exist.Info.Status)
	log.V(5).Info("values diff", "old", exist.Config, "new", values)
	upgrade := action.NewUpgrade(config)
	upgrade.Install = true
	upgrade.Namespace = namespace
	upgradeRelease, err := upgrade.Run(name, chart, values)
	if err != nil {
		log.Error(err, "on upgrade")
		return err
	}
	log.Info("upgraded")
	updateStatusFromRelease(status, upgradeRelease)
	return nil
}

func updateStatusFromRelease(status *pluginsv1beta1.InstallerStatusStatus, release *release.Release) {
	if release == nil {
		return
	}
	resetStatus(status)

	status.Status = pluginsv1beta1.Status(release.Info.Status)
	status.Message = release.Info.Description

	// https://github.com/golang/go/issues/19502
	// metav1.Time and time.Time are not comparable directly
	convtime := func(t time.Time) metav1.Time {
		t, _ = time.Parse(time.RFC3339, t.Format(time.RFC3339))
		return metav1.Time{Time: t}
	}

	status.CreationTimestamp = convtime(release.Info.FirstDeployed.Time)
	status.UpgradeTimestamp = convtime(release.Info.LastDeployed.Time)
	status.Notes = release.Info.Notes
	status.Version = release.Chart.Metadata.Version
	status.Values = marshalValues(release.Config)

	if !release.Info.Deleted.IsZero() {
		status.DeletionTimestamp = func() *metav1.Time {
			time := convtime(release.Info.Deleted.Time)
			return &time
		}()
	}
}

func resetStatus(status *pluginsv1beta1.InstallerStatusStatus) {
	*status = pluginsv1beta1.InstallerStatusStatus{Name: status.Name, Namespace: status.Namespace}
}

func extractCurrentStatus(plugin pluginsv1beta1.InstallerSpecPlugin, installer *pluginsv1beta1.Installer) *pluginsv1beta1.InstallerStatusStatus {
	for i, exist := range installer.Status.States {
		if exist.Name != plugin.Name || exist.Namespace != plugin.Namespace {
			continue
		}
		return &installer.Status.States[i]
	}
	installer.Status.States = append(installer.Status.States, pluginsv1beta1.InstallerStatusStatus{
		Name:      plugin.Name,
		Namespace: plugin.Namespace,
	})
	status := &installer.Status.States[len(installer.Status.States)-1]
	return status
}

func updatedStatus(
	plugin pluginsv1beta1.InstallerSpecPlugin,
	installer *pluginsv1beta1.Installer,
	status *pluginsv1beta1.InstallerStatusStatus,
) bool {
	name, namespace := plugin.Name, plugin.Namespace
	if status != nil {
		status.Name = name
		status.Namespace = namespace
	}
	for i, exist := range installer.Status.States {
		if exist.Name != name || exist.Namespace != namespace {
			continue
		}
		if status == nil {
			installer.Status.States = append(installer.Status.States[:i], installer.Status.States[i+1:]...)
			return true
		}
		if reflect.DeepEqual(exist, *status) {
			return false
		}
		installer.Status.States[i] = *status
		return true
	}
	if status != nil {
		installer.Status.States = append(installer.Status.States, *status)
		return true
	}
	return false
}

func marshalValues(vals map[string]interface{}) runtime.RawExtension {
	if vals == nil {
		return runtime.RawExtension{}
	}
	bytes, _ := json.Marshal(vals)
	return runtime.RawExtension{Raw: bytes}
}

func unmarshalValues(val runtime.RawExtension) map[string]interface{} {
	if val.Raw == nil {
		return nil
	}
	var vals interface{}
	_ = yaml.Unmarshal(val.Raw, &vals)

	if kvs, ok := vals.(map[string]interface{}); ok {
		return kvs
	}
	if arr, ok := vals.([]interface{}); ok {
		// is format of --set K=V
		kvs := make(map[string]interface{}, len(arr))
		for _, kv := range arr {
			if kv, ok := kv.(map[string]interface{}); ok {
				for k, v := range kv {
					kvs[k] = v
				}
			}
		}
		return kvs
	}
	return nil
}

func NewHelmConfig(namespace string) *action.Configuration {
	getter := genericclioptions.NewConfigFlags(true)
	getter.Namespace = pointer.String(namespace) // must set to ns to install chart
	config := &action.Configuration{}
	config.Init(getter, namespace, "", func(format string, v ...interface{}) {
	})
	return config
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstallerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pluginsv1beta1.Installer{}).
		Complete(r)
}
