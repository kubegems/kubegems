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

package plugin

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	PluginFinalizerName             = "plugins.kubegems.io/finalizer"
	PluginAnnotationsShowFullValues = "plugins.kubegems.io/show-full-values"
)

// Reconciler reconciles a Memcached object
type Reconciler struct {
	client.Client
	Applier *PluginApplier
}

type Options struct {
	Cache      string
	SearchDirs []string
}

func NewDefaultOptions() *Options {
	return &Options{}
}

func SetupReconciler(ctx context.Context, mgr manager.Manager, options *Options) error {
	reconciler := &Reconciler{
		Client:  mgr.GetClient(),
		Applier: NewApplier(mgr.GetConfig(), mgr.GetClient(), options),
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&pluginsv1beta1.Plugin{}).
		// WithOptions(controller.Options{}).
		Complete(reconciler)
}

//+kubebuilder:rbac:groups=kubegems.io,resources=installers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubegems.io,resources=installers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubegems.io,resources=installers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// -  https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContextOrDiscard(ctx)

	plugin := &pluginsv1beta1.Plugin{}
	if err := r.Client.Get(ctx, req.NamespacedName, plugin); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer.
	if plugin.DeletionTimestamp == nil && !controllerutil.ContainsFinalizer(plugin, PluginFinalizerName) {
		log.Info("add finalizer", "finalizer", PluginFinalizerName)
		controllerutil.AddFinalizer(plugin, PluginFinalizerName)
		if err := r.Update(ctx, plugin); err != nil {
			return ctrl.Result{}, err
		}
	}

	// check the object is being deleted then remove the finalizer
	if plugin.DeletionTimestamp != nil && controllerutil.ContainsFinalizer(plugin, PluginFinalizerName) {
		if plugin.Status.Phase == pluginsv1beta1.PluginPhaseNone || plugin.Status.Phase == "" {
			log.Info("remvove finalizer", "finalizer", PluginFinalizerName)
			controllerutil.RemoveFinalizer(plugin, PluginFinalizerName)
			if err := r.Update(ctx, plugin); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		log.Info("waiting for plugin to be removed, then remove finalizer")
	}

	beforePlugin := plugin.DeepCopy()
	err := r.Sync(ctx, plugin)
	if err := r.Status().Update(ctx, plugin.DeepCopy()); err != nil {
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(plugin.Spec.Values.Object, beforePlugin.Spec.Values.Object) {
		valbytes, err := plugin.Spec.Values.MarshalJSON()
		if err != nil {
			return ctrl.Result{}, err
		}
		patch := client.RawPatch(types.MergePatchType, []byte(`{"spec":{"values":`+string(valbytes)+`}}`))
		if err := r.Patch(ctx, plugin, patch); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, err
}

type DependencyError struct {
	Reason     string
	Dependency pluginsv1beta1.Dependency
}

func (e DependencyError) Error() string {
	return fmt.Sprintf("dependency %s/%s :%s", e.Dependency.Namespace, e.Dependency.Name, e.Reason)
}

// Sync
func (r *Reconciler) Sync(ctx context.Context, plugin *pluginsv1beta1.Plugin) error {
	shouldRemove := (!plugin.Spec.Enabled) || plugin.DeletionTimestamp != nil

	// todo: check dependencies
	// nolint: nestif
	if !shouldRemove && len(plugin.Spec.Dependencies) > 0 {
		// check dependencies are installed
		for _, dep := range plugin.Spec.Dependencies {
			name, namespace, version := dep.Name, dep.Namespace, dep.Version
			if namespace == "" {
				namespace = plugin.Namespace
			}
			if name == "" {
				continue
			}
			depPlugin := &pluginsv1beta1.Plugin{}
			if err := r.Client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, depPlugin); err != nil {
				if apierrors.IsNotFound(err) {
					return DependencyError{Reason: "not found", Dependency: dep}
				}
				return err
			}
			if depPlugin.Status.Phase != pluginsv1beta1.PluginPhaseInstalled {
				return DependencyError{Reason: "not installed", Dependency: dep}
			}
			if version != "" {
				// TODO: check version
			}
		}
	}

	if shouldRemove {
		// remove
		return r.Applier.Remove(ctx, plugin)
	} else {
		// apply
		return r.Applier.Apply(ctx, plugin)
	}
}
