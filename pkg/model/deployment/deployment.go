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
	"fmt"

	"github.com/go-logr/logr"
	oamv1beta1 "github.com/oam-dev/kubevela/apis/core.oam.dev/v1beta1"
	machinelearningv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
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
		// check owner
		if !IsOwnByModelDeployment(obj) {
			return nil
		}
		return []reconcile.Request{{NamespacedName: client.ObjectKeyFromObject(obj)}}
	})
}

func IsOwnByModelDeployment(obj client.Object) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.APIVersion == modelsv1beta1.GroupVersion.String() && ref.Kind == "ModelDeployment" {
			return true
		}
	}
	return false
}

func Setup(ctx context.Context, mgr ctrl.Manager, options *Options) error {
	r := &Reconciler{
		Client:  mgr.GetClient(),
		Options: options,
		ModelServes: map[string]ModelServe{
			// OAMModelServeKind:    &OAMModelServe{Client: mgr.GetClient()},
			SeldonModelServeKind: &SeldonModelServe{Client: mgr.GetClient()},
		},
	}
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&modelsv1beta1.ModelDeployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: ControllerConcurrency})
	for _, serve := range r.ModelServes {
		builder.Watches(&source.Kind{Type: serve.Watches()}, OAMAppTrigger())
	}
	return builder.Complete(r)
}

type Reconciler struct {
	client.Client
	Options     *Options
	ModelServes map[string]ModelServe
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

	origin := md.DeepCopy()
	if err := r.Default(ctx, md); err != nil {
		return ctrl.Result{}, err
	}
	if !equality.Semantic.DeepEqual(md, origin) {
		if err := r.Client.Update(ctx, md); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := r.Sync(ctx, md); err != nil {
		log.Error(err, "sync model deployment")
		md.Status.Phase = modelsv1beta1.Failed
		md.Status.Message = err.Error()
	}
	if err := r.Status().Update(ctx, md); err != nil {
		return ctrl.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *Reconciler) Sync(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	// apply md
	serve, ok := r.ModelServes[md.Spec.Backend]
	if !ok {
		return fmt.Errorf("unsupported model deployment kind: %s", md.Spec.Backend)
	}
	if err := serve.Apply(ctx, md); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) Default(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	if md.Spec.Host == "" {
		md.Spec.Host = RandStringRunes(r.Options.RandSubDomainLen) + "." + r.Options.BaseDomain
	}
	if md.Spec.Backend == "" {
		md.Spec.Backend = SeldonModelServeKind
	}
	return nil
}
