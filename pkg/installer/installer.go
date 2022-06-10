package installer

import (
	"context"
	"strings"
	"unsafe"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	bundlev1 "kubegems.io/bundle-controller/pkg/apis/bundle/v1beta1"
	"kubegems.io/bundle-controller/pkg/bundle"
	"kubegems.io/bundle-controller/pkg/controllers"
	"kubegems.io/kubegems/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
		BundleReconciler: &controllers.BundleReconciler{
			Client:  mgr.GetClient(),
			Applier: bundle.NewDefaultApply(mgr.GetConfig(), mgr.GetClient(), options),
		},
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
	*controllers.BundleReconciler
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
		if plugin.Status.Phase == bundlev1.PhaseFailed || plugin.Status.Phase == bundlev1.PhaseDisabled {
			controllerutil.RemoveFinalizer(plugin, FinalizerName)
			if err := r.Update(ctx, plugin); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		log.Info("waiting for app to be removed, then remove finalizer")
	}

	err := r.BundleReconciler.Sync(ctx, (*bundlev1.Bundle)(unsafe.Pointer(plugin)))
	if err != nil {
		plugin.Status.Phase = bundlev1.PhaseFailed
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
