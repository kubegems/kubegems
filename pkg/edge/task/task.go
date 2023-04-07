package task

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"kubegems.io/kubegems/pkg/apis/edge"
	edgev1beta1 "kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

func getScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	edgev1beta1.AddToScheme(scheme)
	return scheme
}

type Options struct {
	MaxConcurrentReconciles int
	EdgeServerAddr          string
}

func NewDefaultOptions() *Options {
	return &Options{
		MaxConcurrentReconciles: 1,
		EdgeServerAddr:          "http://kubegem-edge-server:8080",
	}
}

func Run(ctx context.Context, options *Options) error {
	log := log.LogrLogger
	ctx = logr.NewContext(ctx, log)
	ctrl.SetLogger(log)

	log = log.WithName("setup")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:           getScheme(),
		LeaderElectionID: edge.GroupName + "-task",
		Logger:           log,
	})
	if err != nil {
		log.Error(err, "unable to create manager")
		return err
	}
	holder, err := NewEdgeClientsHolder(ctx, options.EdgeServerAddr)
	if err != nil {
		log.Error(err, "unable to create edge clients holder")
		return err
	}
	r := &Reconciler{
		Client:      mgr.GetClient(),
		EdgeClients: holder,
	}
	if err := r.SetupWithManager(ctx, mgr, options.MaxConcurrentReconciles); err != nil {
		log.Error(err, "unable to create controller", "controller", "EdgeTask")
	}
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up ready check")
		return err
	}
	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem running manager")
		return err
	}
	return nil
}
