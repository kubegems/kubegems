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

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
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

	original := installer.DeepCopy()

	for _, plugin := range installer.Spec.Plugins {
		r.sync(ctx, plugin, installer)
	}

	// check has error
	var err error
	// update status
	if !reflect.DeepEqual(original.Status, installer.Status) {
		installer.Status.LastReconcileTime = metav1.Now()
		err = r.Update(ctx, installer)
	}
	for _, status := range installer.Status.States {
		if status.Status != release.StatusDeployed && !status.Status.IsPending() {
			err = errors.New(status.Message)
			break
		}
	}

	log.Info("reconciled", "err", err)
	return err
}

func (r *InstallerReconciler) sync(ctx context.Context, plugin pluginsv1beta1.InstallerSpecPlugin, installer *pluginsv1beta1.Installer) {
	namespace, name, enabled := plugin.Namespace, plugin.Name, plugin.Enabled

	if name == "" || namespace == "" {
		return
	}

	log := logr.FromContext(ctx)

	var desireRelease *release.Release
	var err error
	if enabled {
		desireRelease, err = r.convertRelease(plugin)
		if err != nil {
			setStatus(name, namespace, installer, &pluginsv1beta1.InstallerStatusStatus{
				Status:  release.StatusFailed,
				Message: err.Error(),
			})
			return
		}
	}

	existrelease, err := action.NewGet(NewConfig(namespace)).Run(name)
	if err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			setStatus(name, namespace, installer, &pluginsv1beta1.InstallerStatusStatus{
				Status:  release.StatusFailed,
				Message: err.Error(),
			})
		}
	}

	status := r.apply(ctx, existrelease, desireRelease)
	if status == nil {
		return
	}

	if status.Status != release.StatusDeployed {
		log.Error(errors.New(status.Message), "apply", "release", name)
	} else {
		log.Info(status.Message, "release", name)
	}

	setStatus(name, namespace, installer, status)
}

// nolint: funlen
func (r *InstallerReconciler) apply(ctx context.Context, exist, desired *release.Release) *pluginsv1beta1.InstallerStatusStatus {
	log := logr.FromContext(ctx)

	switch {
	case exist == nil && desired != nil:
		log.Info("install", "name", desired.Name)
		// install
		install := action.NewInstall(NewConfig(desired.Namespace))
		install.ReleaseName = desired.Name
		install.CreateNamespace = true
		install.Namespace = desired.Namespace
		installedRelease, err := install.Run(desired.Chart, desired.Config)
		if err != nil {
			return &pluginsv1beta1.InstallerStatusStatus{
				Status:  release.StatusFailed,
				Message: err.Error(),
			}
		}
		return &pluginsv1beta1.InstallerStatusStatus{
			Status:            installedRelease.Info.Status,
			Message:           installedRelease.Info.Description,
			CreationTimestamp: metav1.Time(installedRelease.Info.FirstDeployed),
			UpgradeTimestamp:  metav1.Time(installedRelease.Info.LastDeployed),
			Version:           desired.Chart.Metadata.Version,
			Notes:             installedRelease.Info.Notes,
			Values:            marshalValues(installedRelease.Config),
		}
	case exist != nil && desired == nil:
		// uninstall
		log.Info("uninstall", "name", exist.Name)
		uninstall := action.NewUninstall(NewConfig(exist.Namespace))
		uninstalledRelease, err := uninstall.Run(exist.Name)
		if err != nil {
			if errors.Is(err, driver.ErrReleaseNotFound) {
				return &pluginsv1beta1.InstallerStatusStatus{
					Status: release.StatusUninstalled,
				}
			}
			return &pluginsv1beta1.InstallerStatusStatus{
				Status:  release.StatusFailed,
				Message: err.Error(),
			}
		}
		if uninstalledRelease.Release != nil {
			return &pluginsv1beta1.InstallerStatusStatus{
				Status:            uninstalledRelease.Release.Info.Status,
				Message:           uninstalledRelease.Release.Info.Description,
				Values:            marshalValues(uninstalledRelease.Release.Config),
				CreationTimestamp: metav1.Time(uninstalledRelease.Release.Info.FirstDeployed),
				UpgradeTimestamp:  metav1.Time(uninstalledRelease.Release.Info.LastDeployed),
				Version:           uninstalledRelease.Release.Chart.Metadata.Version,
				Notes:             uninstalledRelease.Release.Info.Notes,
				DeletionTimestamp: func() *metav1.Time {
					time := metav1.Time(uninstalledRelease.Release.Info.Deleted)
					return &time
				}(),
			}
		}
		return &pluginsv1beta1.InstallerStatusStatus{
			Status:  release.StatusUnknown,
			Message: uninstalledRelease.Info,
		}
	case exist != nil && desired != nil:
		// upgrade
		if (len(exist.Config) == 0 && len(desired.Config) == 0) || reflect.DeepEqual(exist.Config, desired.Config) {
			return nil
		}
		log.Info("upgrade", "name", desired.Name)
		upgrade := action.NewUpgrade(NewConfig(desired.Namespace))
		upgrade.Namespace = desired.Namespace
		upgradeRelease, err := upgrade.Run(desired.Name, desired.Chart, desired.Config)
		if err != nil {
			return &pluginsv1beta1.InstallerStatusStatus{
				Status:  release.StatusFailed,
				Message: err.Error(),
			}
		}
		return &pluginsv1beta1.InstallerStatusStatus{
			Status:            upgradeRelease.Info.Status,
			Message:           upgradeRelease.Info.Description,
			CreationTimestamp: metav1.Time(upgradeRelease.Info.FirstDeployed),
			UpgradeTimestamp:  metav1.Time(upgradeRelease.Info.LastDeployed),
			Notes:             upgradeRelease.Info.Notes,
			Version:           desired.Chart.Metadata.Version,
			Values:            marshalValues(upgradeRelease.Config),
		}
	}
	return nil
}

func (r *InstallerReconciler) convertRelease(plugin pluginsv1beta1.InstallerSpecPlugin) (*release.Release, error) {
	pluginValues := map[string]interface{}{}
	_ = json.Unmarshal(plugin.Values.Raw, &pluginValues)
	loader, err := loader.Loader(filepath.Join(r.Options.ChartsDir, plugin.Name))
	if err != nil {
		return nil, err
	}
	chart, err := loader.Load()
	if err != nil {
		return nil, err
	}
	return &release.Release{
		Name:      plugin.Name,
		Namespace: plugin.Namespace,
		Config:    pluginValues,
		Chart:     chart,
	}, nil
}

func setStatus(name string, namespace string, installer *pluginsv1beta1.Installer, status *pluginsv1beta1.InstallerStatusStatus) {
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
			return
		}
		installer.Status.States[i] = *status
		return
	}
	if status != nil {
		installer.Status.States = append(installer.Status.States, *status)
	}
}

func marshalValues(vals map[string]interface{}) runtime.RawExtension {
	bytes, _ := yaml.Marshal(vals)
	return runtime.RawExtension{Raw: bytes}
}

func NewConfig(namespace string) *action.Configuration {
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
