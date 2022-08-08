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

package deployment

import (
	"context"

	"github.com/go-logr/logr"
	oamv1beta1 "github.com/oam-dev/kubevela/apis/core.oam.dev/v1beta1"
	machinelearningv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/apis/models"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerConcurrency = 5
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
	machinelearningv1.AddToScheme(scheme)
	oamv1beta1.AddToScheme(scheme)
	gemsv1beta1.AddToScheme(scheme)
}

type Options struct {
	MetricsAddr          string `json:"metricsAddr,omitempty" description:"The address the metric endpoint binds to."`
	EnableLeaderElection bool   `json:"enableLeaderElection,omitempty" description:"Enable leader election for controller manager."`
	ProbeAddr            string `json:"probeAddr,omitempty" description:"The address the probe endpoint binds to."`
	IngressHost          string `json:"ingressHost,omitempty" description:"The base host of the ingress."`
	IngressScheme        string `json:"ingressScheme,omitempty" description:"The scheme of the ingress."`
}

func DefaultOptions() *Options {
	return &Options{
		MetricsAddr:          "127.0.0.1:9100", // default run under kube-rbac-proxy
		EnableLeaderElection: false,
		ProbeAddr:            ":8081",
		IngressHost:          "models.kubegems.io",
		IngressScheme:        "http",
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
		if IsOwnBy(obj, modelsv1beta1.GroupVersion.WithKind("ModelDeployment")) {
			return []reconcile.Request{{NamespacedName: client.ObjectKeyFromObject(obj)}}
		}
		return nil
	})
}

func IsOwnBy(obj client.Object, gvk schema.GroupVersionKind) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.APIVersion == gvk.GroupVersion().String() && ref.Kind == gvk.Kind {
			return true
		}
	}
	return false
}

func Setup(ctx context.Context, mgr ctrl.Manager, options *Options) error {
	r := &Reconciler{
		Client:  mgr.GetClient(),
		Options: options,
		SeldonBack: &SeldonModelServe{
			Client:        mgr.GetClient(),
			IngressHost:   options.IngressHost,
			IngressScheme: options.IngressScheme,
		},
	}
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&modelsv1beta1.ModelDeployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: ControllerConcurrency}).
		Watches(&source.Kind{Type: &machinelearningv1.SeldonDeployment{}}, OAMAppTrigger())

	return builder.Complete(r)
}

type Reconciler struct {
	client.Client
	Options    *Options
	SeldonBack *SeldonModelServe
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

	// use default on next process,but do not writeback.
	if err := r.Default(ctx, md); err != nil {
		log.Error(err, "failed to default model deployment")
		return ctrl.Result{}, err
	}

	if err := r.Sync(ctx, md); err != nil {
		log.Error(err, "sync model deployment")
		md.Status.Phase = modelsv1beta1.Failed
		md.Status.Message = err.Error()
	}

	if err := r.updateStatus(ctx, md); err != nil {
		return ctrl.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *Reconciler) updateStatus(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	existing := &modelsv1beta1.ModelDeployment{ObjectMeta: md.ObjectMeta}
	if err := r.Get(ctx, client.ObjectKeyFromObject(existing), existing); err != nil {
		return err
	}
	if equality.Semantic.DeepEqual(md.Status, existing.Status) {
		return nil
	}
	existing.Status = md.Status
	return r.Status().Update(ctx, existing)
}

func (r *Reconciler) Sync(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	return r.SeldonBack.Apply(ctx, md)
}

func (r *Reconciler) Default(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	return nil
}
