package installer

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"kubegems.io/pkg/apis/plugins"
	pluginv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"kubegems.io/pkg/installer/controllers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(pluginv1beta1.AddToScheme(scheme))
}

type Options struct {
	MetricsAddr          string `json:"metricsAddr,omitempty" description:"The address the metric endpoint binds to."`
	EnableLeaderElection bool   `json:"enableLeaderElection,omitempty" description:"Enable leader election for controller manager."`
	ProbeAddr            string `json:"probeAddr,omitempty" description:"The address the probe endpoint binds to."`
}

func NewDefaultOptions() *Options {
	return &Options{
		MetricsAddr:          "127.0.0.1:8080", // default run under kube-rbac-proxy
		EnableLeaderElection: false,
		ProbeAddr:            ":8081",
	}
}

func Run(ctx context.Context, options *Options) error {
	ctrl.SetLogger(zap.New(zap.UseDevMode(false)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     options.MetricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: options.ProbeAddr,
		LeaderElection:         options.EnableLeaderElection,
		LeaderElectionID:       plugins.GroupName,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	installerctrl := &controllers.InstallerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Options: &controllers.InstallerOptions{
			ChartsDir: "charts",
		},
	}
	if err = installerctrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Memcached")
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
