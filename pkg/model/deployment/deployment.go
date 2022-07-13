package deployment

import (
	"context"

	"github.com/go-logr/logr"
	oamcommon "github.com/oam-dev/kubevela/apis/core.oam.dev/common"
	oamv1beta1 "github.com/oam-dev/kubevela/apis/core.oam.dev/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/utils/pointer"
	"kubegems.io/kubegems/pkg/apis/models"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
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
	ControllerConcurrency = 5
	FinalizerName         = models.GroupName + "/finalizer"
	DefaultSubdomainLen   = 8
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

// nolint: gochecknoinits
func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(modelsv1beta1.AddToScheme(scheme))
	oamv1beta1.AddToScheme(scheme)
}

type Options struct {
	MetricsAddr          string `json:"metricsAddr,omitempty" description:"The address the metric endpoint binds to."`
	EnableLeaderElection bool   `json:"enableLeaderElection,omitempty" description:"Enable leader election for controller manager."`
	ProbeAddr            string `json:"probeAddr,omitempty" description:"The address the probe endpoint binds to."`
	BaseDomain           string `json:"baseDomain,omitempty" description:"The base domain of the servingmodel"`
	RandSubDomainLen     int    `json:"randSubDomainLen,omitempty" description:"The length of the random sub domain"`
}

func DefaultOptions() *Options {
	return &Options{
		MetricsAddr:          "127.0.0.1:9100", // default run under kube-rbac-proxy
		EnableLeaderElection: false,
		ProbeAddr:            ":8081",
		BaseDomain:           "models.kubegems.io",
		RandSubDomainLen:     DefaultSubdomainLen,
	}
}

func Run(ctx context.Context, options *Options) error {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     options.MetricsAddr,
		HealthProbeBindAddress: options.ProbeAddr,
		LeaderElection:         options.EnableLeaderElection,
		LeaderElectionID:       models.GroupName,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	if err := Setup(ctx, mgr, options); err != nil {
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

func OAMAppTrigger() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: client.ObjectKeyFromObject(obj)}}
	})
}

func Setup(ctx context.Context, mgr ctrl.Manager, options *Options) error {
	r := &Reconciler{
		Client:  mgr.GetClient(),
		Options: options,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&modelsv1beta1.ModelDeployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: ControllerConcurrency}).
		Watches(&source.Kind{Type: &oamv1beta1.Application{}}, OAMAppTrigger()).
		Complete(r)
}

type Reconciler struct {
	client.Client
	Options *Options
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContextOrDiscard(ctx)

	log.Info("reconciling model deployment")
	md := &modelsv1beta1.ModelDeployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, md); err != nil {
		if errors.IsNotFound(err) {
			log.Info("resource not found. ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if md.DeletionTimestamp != nil {
		log.Info("being deleted. ignoring")
		return ctrl.Result{}, nil
	}

	err := r.Sync(ctx, md)
	if err != nil {
		log.Error(err, "sync model deployment")
		md.Status.Phase = modelsv1beta1.Failed
		md.Status.Message = err.Error()
	}
	_ = r.Status().Update(ctx, md)
	return ctrl.Result{}, err
}

func (r *Reconciler) Sync(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	changed, err := r.Fulfill(ctx, md)
	if err != nil {
		return err
	}
	if changed {
		_ = r.Update(ctx, md)
		return nil
	}

	// apply oamapp
	oamapp := &oamv1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      md.Name,
			Namespace: md.Namespace,
		},
	}

	onupdatefun := func() error {
		Mergekvs(md.Annotations, oamapp.Annotations)
		Mergekvs(md.Labels, oamapp.Labels)
		_ = controllerutil.SetOwnerReference(md, oamapp, r.Client.Scheme())
		return r.DeployHuggingFaceModel(ctx, md, oamapp)
	}
	// oam app update frequently,use patch instead of update
	if _, err := controllerutil.CreateOrPatch(ctx, r.Client, oamapp, onupdatefun); err != nil {
		return err
	}

	// fill phase
	md.Status.OAMStatus = oamapp.Status
	md.Status.Message = ""
	md.Status.Phase = func() modelsv1beta1.Phase {
		switch oamapp.Status.Phase {
		case oamcommon.ApplicationRunning:
			return modelsv1beta1.Running
		default:
			return modelsv1beta1.Pending
		}
	}()
	return nil
}

func (r *Reconciler) Fulfill(ctx context.Context, md *modelsv1beta1.ModelDeployment) (bool, error) {
	haschange := false
	if md.Spec.Host == "" {
		md.Spec.Host = RandStringRunes(r.Options.RandSubDomainLen) + "." + r.Options.BaseDomain
		haschange = true
	}
	return haschange, nil
}

func (r *Reconciler) DeployHuggingFaceModel(ctx context.Context, md *modelsv1beta1.ModelDeployment, oamapp *oamv1beta1.Application) error {
	const servingPort = 8080
	oamapp.Spec = oamv1beta1.ApplicationSpec{
		Components: []oamcommon.ApplicationComponent{
			{
				Name: md.Name,
				Type: "webservice",
				Properties: OAMWebServiceProperties{
					Labels:      md.Labels,
					Annotations: md.Annotations,
					Image:       md.Spec.Model.Image,
					ENV: []OAMWebServicePropertiesEnv{
						{
							Name:  "PKG",
							Value: md.Spec.Model.Framework,
						},
						{
							Name:  "MODEL",
							Value: md.Spec.Model.Name,
						},
					},
					Ports: []OAMWebServicePropertiesPort{
						{Name: "http", Port: servingPort},
					},
				}.RawExtension(),
				Traits: []oamcommon.ApplicationTrait{
					{
						Type: "scaler",
						Properties: models.Properties{
							"replicas": pointer.Int32Deref(md.Spec.Replicas, 1),
						}.ToRawExtension(),
					},
					{
						Type: "gateway",
						Properties: models.Properties{
							"domain": md.Spec.Host,
							"http": map[string]interface{}{
								"/": servingPort,
							},
							"classInSpec": true,
						}.ToRawExtension(),
					},
				},
			},
		},
	}
	return nil
}
