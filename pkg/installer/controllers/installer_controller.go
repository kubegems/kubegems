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
	"fmt"

	"github.com/go-logr/logr"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"
)

// InstallerReconciler reconciles a Memcached object
type InstallerReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	Applyers   map[pluginsv1beta1.InstallerSpecPluginKind]Applier
}

type InstallerOptions struct {
	ChartsDir  string `json:"chartsDir,omitempty"`
	PluginsDir string `json:"pluginsDir,omitempty"`
}

func NewAndSetupInstallerReconciler(ctx context.Context, mgr manager.Manager, options *InstallerOptions) error {
	nativeApplier, err := NewNativeApplier(ctx, mgr, options.PluginsDir)
	if err != nil {
		return err
	}
	applyers := map[pluginsv1beta1.InstallerSpecPluginKind]Applier{
		pluginsv1beta1.InstallerSpecPluginKindHelm:   &HelmApplier{ChartsDir: options.ChartsDir},
		pluginsv1beta1.InstallerSpecPluginKindNative: nativeApplier,
	}
	reconciler := &InstallerReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		RestConfig: mgr.GetConfig(),
		Applyers:   applyers,
	}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		return err
	}
	return nil
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

type Applier interface {
	Apply(ctx context.Context, plugin pluginsv1beta1.InstallerSpecPlugin, status *pluginsv1beta1.InstallerStatusStatus) error
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

		// if empty use default
		if plugin.Kind == "" {
			plugin.Kind = pluginsv1beta1.InstallerSpecPluginKindHelm
			continue
		}

		// choose applyer
		applyer, ok := r.Applyers[plugin.Kind]
		if !ok {
			status.Status = pluginsv1beta1.StatusFailed
			status.Message = fmt.Sprintf("unknow plugin kind %s", plugin.Kind)
			if err := r.Status().Update(ctx, installer); err != nil {
				return err
			}
			continue
		}
		if err := applyer.Apply(ctx, plugin, status); err != nil {
			resetStatus(status)
			status.Status = pluginsv1beta1.StatusFailed
			status.Message = err.Error()
			if err := r.Status().Update(ctx, installer); err != nil {
				return err
			}
			return err
		}
		status.Namespace, status.Name, status.Kind = plugin.Namespace, plugin.Name, plugin.Kind

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

func MarshalValues(vals map[string]interface{}) runtime.RawExtension {
	if vals == nil {
		return runtime.RawExtension{}
	}
	bytes, _ := json.Marshal(vals)
	return runtime.RawExtension{Raw: bytes}
}

func UnmarshalValues(val runtime.RawExtension) map[string]interface{} {
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

// SetupWithManager sets up the controller with the Manager.
func (r *InstallerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pluginsv1beta1.Installer{}).
		Complete(r)
}
