// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/strvals"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"kubegems.io/kubegems/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/bundle"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/yaml"
)

const (
	PluginsControllerConcurrency = 5
	FinalizerName                = "plugins.kubegems.io/finalizer"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

// nolint: gochecknoinits
func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(pluginsv1beta1.AddToScheme(scheme))
}

type Options struct {
	MetricsAddr          string `json:"metricsAddr,omitempty" description:"The address the metric endpoint binds to."`
	EnableLeaderElection bool   `json:"enableLeaderElection,omitempty" description:"Enable leader election for controller manager."`
	ProbeAddr            string `json:"probeAddr,omitempty" description:"The address the probe endpoint binds to."`
	PluginsDir           string `json:"pluginsDir,omitempty" description:"The directory that contains the plugins."`
}

func NewDefaultOptions() *Options {
	return &Options{
		MetricsAddr:          "127.0.0.1:9100", // default run under kube-rbac-proxy
		EnableLeaderElection: false,
		ProbeAddr:            ":8081",
		PluginsDir:           "plugins",
	}
}

func Run(ctx context.Context, options *Options) error {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     options.MetricsAddr,
		HealthProbeBindAddress: options.ProbeAddr,
		LeaderElection:         options.EnableLeaderElection,
		LeaderElectionID:       plugins.GroupName,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	bundleoptions := bundle.NewDefaultOptions()
	bundleoptions.SearchDirs = append(bundleoptions.SearchDirs, strings.Split(options.PluginsDir, ",")...)
	if err := Setup(ctx, mgr, bundleoptions); err != nil {
		setupLog.Error(err, "unable to create plugin controller", "controller", "plugin")
		return err
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}
	return nil
}

func Setup(ctx context.Context, mgr ctrl.Manager, options *bundle.Options) error {
	r := &Reconciler{
		Client:  mgr.GetClient(),
		Applier: bundle.NewDefaultApply(mgr.GetConfig(), mgr.GetClient(), options),
	}
	handler := ConfigMapOrSecretTrigger(ctx, mgr.GetClient())
	return ctrl.NewControllerManagedBy(mgr).
		For(&pluginsv1beta1.Plugin{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: PluginsControllerConcurrency}).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, handler).
		Watches(&source.Kind{Type: &corev1.Secret{}}, handler).
		Complete(r)
}

type Reconciler struct {
	client.Client
	Applier *bundle.BundleApplier
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContextOrDiscard(ctx)

	plugin := &pluginsv1beta1.Plugin{}
	if err := r.Client.Get(ctx, req.NamespacedName, plugin); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Plugin resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer.
	if plugin.DeletionTimestamp == nil && !controllerutil.ContainsFinalizer(plugin, FinalizerName) {
		log.Info("add finalizer")
		controllerutil.AddFinalizer(plugin, FinalizerName)
		if err := r.Update(ctx, plugin); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// check the object is being deleted then remove the finalizer
	if plugin.DeletionTimestamp != nil && controllerutil.ContainsFinalizer(plugin, FinalizerName) {
		if plugin.Status.Phase == pluginsv1beta1.PhaseFailed || plugin.Status.Phase == pluginsv1beta1.PhaseDisabled {
			controllerutil.RemoveFinalizer(plugin, FinalizerName)
			if err := r.Update(ctx, plugin); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		log.Info("waiting for app to be removed, then remove finalizer")
	}

	err := r.Sync(ctx, plugin)
	if err != nil {
		plugin.Status.Phase = pluginsv1beta1.PhaseFailed
		plugin.Status.Message = err.Error()
	}

	// update status if updated whenever the sync has error or no
	if err := r.Status().Update(ctx, plugin); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}

func ConfigMapOrSecretTrigger(ctx context.Context, cli client.Client) handler.EventHandler {
	log := logr.FromContextOrDiscard(ctx)
	return handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
		kind := ""
		switch obj.(type) {
		case *corev1.ConfigMap:
			kind = "ConfigMap"
		case *corev1.Secret:
			kind = "Secret"
		default:
			return nil
		}

		plugins := pluginsv1beta1.PluginList{}
		_ = cli.List(ctx, &plugins, client.InNamespace(obj.GetNamespace()))
		var requests []reconcile.Request
		for _, item := range plugins.Items {
			for _, ref := range item.Spec.ValuesFrom {
				if ref.Kind == kind && ref.Name == obj.GetName() {
					log.Info("triggering reconciliation", "plugin", item.Name, "kind", kind, "name", obj.GetName(), "namespace", item.GetNamespace())
					requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&item)})
				}
			}
		}
		return requests
	})
}

// Sync
func (r *Reconciler) Sync(ctx context.Context, bundle *pluginsv1beta1.Plugin) error {
	if bundle.Spec.Disabled || bundle.DeletionTimestamp != nil {
		// just remove
		return r.Applier.Remove(ctx, bundle)
	} else {
		// check all dependencies are installed
		if err := r.checkDepenency(ctx, bundle); err != nil {
			return err
		}
		// resolve valuesRef
		if err := r.resolveValuesRef(ctx, bundle); err != nil {
			return err
		}
		return r.Applier.Apply(ctx, bundle)
	}
}

type DependencyError struct {
	Reason string
	Object corev1.ObjectReference
}

func (e DependencyError) Error() string {
	return fmt.Sprintf("dependency %s/%s :%s", e.Object.Namespace, e.Object.Name, e.Reason)
}

func (r *Reconciler) checkDepenency(ctx context.Context, bundle *pluginsv1beta1.Plugin) error {
	for _, dep := range bundle.Spec.Dependencies {
		if dep.Name == "" {
			continue
		}
		if dep.Namespace == "" {
			dep.Namespace = bundle.Namespace
		}
		if dep.Kind == "" {
			dep.APIVersion = bundle.APIVersion
			dep.Kind = bundle.Kind
		}
		newobj, _ := r.Scheme().New(dep.GroupVersionKind())
		depobj, ok := newobj.(client.Object)
		if !ok {
			depobj = &metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: dep.GroupVersionKind().GroupVersion().String(),
					Kind:       dep.Kind,
				},
			}
		}

		// exists check
		if err := r.Client.Get(ctx, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name}, depobj); err != nil {
			if apierrors.IsNotFound(err) {
				return DependencyError{Reason: err.Error(), Object: dep}
			}
			return err
		}

		// status check
		switch obj := depobj.(type) {
		case *pluginsv1beta1.Plugin:
			if obj.Status.Phase != pluginsv1beta1.PhaseInstalled {
				return DependencyError{Reason: "not installed", Object: dep}
			}
		}
	}
	return nil
}

func (r *Reconciler) resolveValuesRef(ctx context.Context, bundle *pluginsv1beta1.Plugin) error {
	base := map[string]interface{}{}

	for _, ref := range bundle.Spec.ValuesFrom {
		switch strings.ToLower(ref.Kind) {
		case "secret", "secrets":
			secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: ref.Name, Namespace: bundle.Namespace}}
			if err := r.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
				if ref.Optional && apierrors.IsNotFound(err) {
					continue
				}
				return err
			}
			// --set
			for k, v := range secret.Data {
				if err := mergeInto(ref.Prefix+k, string(v), base); err != nil {
					return fmt.Errorf("parse %#v key[%s]: %w", ref, k, err)
				}
			}
		case "configmap", "configmaps":
			configmap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: ref.Name, Namespace: bundle.Namespace}}
			if err := r.Client.Get(ctx, client.ObjectKeyFromObject(configmap), configmap); err != nil {
				if ref.Optional && apierrors.IsNotFound(err) {
					continue
				}
				return err
			}
			// -f/--values
			for k, v := range configmap.BinaryData {
				currentMap := map[string]interface{}{}
				if err := yaml.Unmarshal(v, &currentMap); err != nil {
					return fmt.Errorf("parse %#v key[%s]: %w", ref, k, err)
				}
				base = mergeMaps(base, currentMap)
			}
			// --set
			for k, v := range configmap.Data {
				if err := mergeInto(ref.Prefix+k, string(v), base); err != nil {
					return fmt.Errorf("parse %#v key[%s]: %w", ref, k, err)
				}
			}
		default:
			return fmt.Errorf("valuesRef kind [%s] is not supported", ref.Kind)
		}
	}

	// inlined values
	base = mergeMaps(base, bundle.Spec.Values.Object)

	bundle.Spec.Values = pluginsv1beta1.Values{Object: base}.FullFill()
	return nil
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func mergeInto(k, v string, base map[string]interface{}) error {
	if err := strvals.ParseInto(fmt.Sprintf("%s=%s", k, v), base); err != nil {
		return fmt.Errorf("parse %#v key[%s]: %w", k, v, err)
	}
	return nil
}
