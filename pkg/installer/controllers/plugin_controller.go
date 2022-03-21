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
	"fmt"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Applier interface {
	Apply(ctx context.Context, plugin Plugin, status *PluginStatus) error
	Remove(ctx context.Context, plugin Plugin, status *PluginStatus) error
}

type Plugin struct {
	Name      string
	Namespace string
	Version   string
	Repo      string
	Values    map[string]interface{}
}

type PluginStatus struct {
	Name              string
	Namespace         string
	Phase             pluginsv1beta1.PluginPhase
	Values            map[string]interface{}
	Version           string
	Message           string
	Notes             string
	CreationTimestamp metav1.Time
	UpgradeTimestamp  metav1.Time
	DeletionTimestamp metav1.Time
}

func PluginStatusFromPlugin(plugin *pluginsv1beta1.Plugin) *PluginStatus {
	if plugin == nil {
		return nil
	}
	return &PluginStatus{
		Name:              plugin.Name,
		Namespace:         plugin.Status.InstallNamespace,
		Phase:             plugin.Status.Phase,
		Message:           plugin.Status.Message,
		Values:            UnmarshalValues(plugin.Status.Values),
		Version:           plugin.Status.Version,
		Notes:             plugin.Status.Notes,
		CreationTimestamp: plugin.CreationTimestamp,
		UpgradeTimestamp:  plugin.Status.UpgradeTimestamp,
		DeletionTimestamp: func() metav1.Time {
			if plugin.DeletionTimestamp.IsZero() {
				return metav1.Time{}
			}
			return *plugin.DeletionTimestamp
		}(),
	}
}

func (s PluginStatus) toPluginStatus() pluginsv1beta1.PluginStatus {
	return pluginsv1beta1.PluginStatus{
		Phase:             s.Phase,
		Message:           s.Message,
		Notes:             s.Notes,
		InstallNamespace:  s.Namespace,
		Values:            MarshalValues(s.Values),
		Version:           s.Version,
		CreationTimestamp: s.CreationTimestamp,
		UpgradeTimestamp:  s.UpgradeTimestamp,
		DeletionTimestamp: func() *metav1.Time {
			if s.DeletionTimestamp.IsZero() {
				return nil
			}
			return &s.DeletionTimestamp
		}(),
	}
}

// PluginReconciler reconciles a Memcached object
type PluginReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	Applyers   map[pluginsv1beta1.PluginKind]Applier
}

type PluginOptions struct {
	ChartsDir  string `json:"chartsDir,omitempty"`
	PluginsDir string `json:"pluginsDir,omitempty"`
}

func NewAndSetupPluginReconciler(ctx context.Context, mgr manager.Manager, options *PluginOptions) error {
	nativeApplier, err := NewNativeApplier(ctx, mgr, options.PluginsDir)
	if err != nil {
		return err
	}
	_ = nativeApplier
	applyers := map[pluginsv1beta1.PluginKind]Applier{
		pluginsv1beta1.PluginKindHelm:   &HelmApplier{ChartsDir: options.ChartsDir},
		pluginsv1beta1.PluginKindNative: nativeApplier,
	}
	reconciler := &PluginReconciler{
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
func (r *PluginReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	plugin := &pluginsv1beta1.Plugin{}
	if err := r.Client.Get(ctx, req.NamespacedName, plugin); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.Sync(ctx, plugin); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// Sync
// nolint: funlen
func (r *PluginReconciler) Sync(ctx context.Context, plugin *pluginsv1beta1.Plugin) error {
	thisPlugin := Plugin{
		Name:    plugin.Name,
		Values:  UnmarshalValues(plugin.Spec.Values),
		Version: plugin.Spec.Version,
		Repo:    plugin.Spec.Repo,
		Namespace: func() string {
			if plugin.Spec.InstallNamespace == "" {
				return plugin.Namespace
			}
			return plugin.Spec.InstallNamespace
		}(),
	}
	thisStatus := PluginStatusFromPlugin(plugin)

	// todo: check dependencies
	if len(plugin.Spec.Dependencies) > 0 {
		// check dependencies are installed
		for _, dep := range plugin.Spec.Dependencies {
			depPlugin := &pluginsv1beta1.Plugin{}
			if err := r.Client.Get(ctx, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, depPlugin); err != nil {
				return err
			}
			if depPlugin.Status.Phase != pluginsv1beta1.PluginPhaseInstalled {
				return fmt.Errorf("dependency %s/%s is not installed", depPlugin.Namespace, depPlugin.Name)
			}
		}
	}

	// choose applyer
	if plugin.Spec.Kind == "" {
		plugin.Spec.Kind = pluginsv1beta1.PluginKindHelm
	}
	applyer, ok := r.Applyers[plugin.Spec.Kind]
	if !ok {
		plugin.Status.Phase = pluginsv1beta1.PluginPhaseFailed
		plugin.Status.Message = fmt.Sprintf("unknow plugin kind %s", plugin.Spec.Kind)
		if err := r.Status().Update(ctx, plugin); err != nil {
			return err
		}
	}

	// nolint: nestif
	if !plugin.Spec.Enabled || plugin.DeletionTimestamp != nil {
		// remove
		if err := applyer.Remove(ctx, thisPlugin, thisStatus); err != nil {
			plugin.Status.Phase = pluginsv1beta1.PluginPhaseFailed
			plugin.Status.Message = err.Error()
			if err := r.Status().Update(ctx, plugin); err != nil {
				return err
			}
			return err
		}
	} else {
		// apply
		if err := applyer.Apply(ctx, thisPlugin, thisStatus); err != nil {
			plugin.Status.Phase = pluginsv1beta1.PluginPhaseFailed
			plugin.Status.Message = err.Error()
			if err := r.Status().Update(ctx, plugin); err != nil {
				return err
			}
			return err
		}
	}

	// update status
	pluginStatus := thisStatus.toPluginStatus()
	if apiequality.Semantic.DeepEqual(plugin.Status, pluginStatus) {
		return nil
	}
	plugin.Status = pluginStatus
	if err := r.Status().Update(ctx, plugin); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PluginReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pluginsv1beta1.Plugin{}).
		Complete(r)
}
