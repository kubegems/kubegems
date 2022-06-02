/*
Copyright 2022 The kubegems.io Authors.

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
package controller

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	loggingv1beta1 "github.com/banzaicloud/logging-operator/pkg/sdk/api/v1beta1"
	"github.com/go-logr/logr"
	nginx_v1alpha1 "github.com/nginxinc/nginx-ingress-operator/api/v1alpha1"
	istioclinetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"kubegems.io/pkg/apis/gems"
	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
	gemscontroller "kubegems.io/pkg/controller/controllers"
	"kubegems.io/pkg/controller/webhooks"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var (
	scheme        = runtime.NewScheme()
	setupLog      = ctrl.Log.WithName("setup")
	leaseDuration = 30 * time.Second
	renewDeadline = 20 * time.Second
)

// nolint: gochecknoinits
func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gemsv1beta1.AddToScheme(scheme))
	utilruntime.Must(loggingv1beta1.AddToScheme(scheme))
	utilruntime.Must(nginx_v1alpha1.SchemeBuilder.AddToScheme(scheme))
	utilruntime.Must(istioclinetworkingv1beta1.SchemeBuilder.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

type Options struct {
	MetricsAddr          string `json:"metricsAddr,omitempty" description:"The address the metric endpoint binds to."`
	ProbeAddr            string `json:"probeAddr,omitempty" description:"The address the probe endpoint binds to."`
	WebhookAddr          string `json:"webhookAddr,omitempty" description:"The address the webhook endpoint binds to."`
	EnableLeaderElection bool   `json:"enableLeaderElection,omitempty" description:"Enable leader election for controller manager."`
	Enablewebhook        bool   `json:"enablewebhook,omitempty" description:"Enable webhook for controller manager."`
	Repository           string `json:"repository,omitempty" description:"default image repo."`
}

func NewDefaultOptions() *Options {
	return &Options{
		WebhookAddr:          ":9443",
		MetricsAddr:          ":9090",
		ProbeAddr:            ":8081",
		EnableLeaderElection: false,
		Enablewebhook:        true,
		Repository:           "docker.io/kubegems",
	}
}

func Run(ctx context.Context, options *Options) error {
	ctrl.SetLogger(zap.New(zap.UseDevMode(false)))

	webhookHost, webhookPortStr, err := net.SplitHostPort(options.WebhookAddr)
	if err != nil {
		return fmt.Errorf("parse webhook address: %v", err)
	}
	webhookPort, err := strconv.Atoi(webhookPortStr)
	if err != nil {
		return fmt.Errorf("parse webhook port: %v", err)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     options.MetricsAddr,
		Port:                   webhookPort,
		Host:                   webhookHost,
		HealthProbeBindAddress: options.ProbeAddr,
		LeaseDuration:          &leaseDuration,
		RenewDeadline:          &renewDeadline,
		LeaderElection:         options.EnableLeaderElection,
		LeaderElectionID:       gems.GroupName,
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		return err
	}
	// setup healthz
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}
	// setup controllers
	if err := setupControllers(mgr, options, setupLog); err != nil {
		return err
	}
	// setup webhooks
	if options.Enablewebhook {
		if err := setUpWebhook(mgr, setupLog, options.Repository); err != nil {
			return err
		}
	}

	go webhooks.CreateDefaultTenantGateway(mgr.GetClient(), ctrl.Log.WithName("create-default-tenant-gateway"))

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}
	return nil
}

func setupControllers(mgr ctrl.Manager, options *Options, setupLog logr.Logger) error {
	if err := (&gemscontroller.TenantReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("Tenant"),
		Recorder: mgr.GetEventRecorderFor("Tenant"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Tenant")
		return err
	}
	if err := (&gemscontroller.TenantResourceQuotaReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TenantResourceQuota")
		return err
	}
	if err := (&gemscontroller.TenantNetworkPolicyReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("TenantNetworkPolicy"),
		Recorder: mgr.GetEventRecorderFor("TenantNetworkPolicy"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TenantNetworkPolicy")
		return err
	}
	if err := (&gemscontroller.TenantGatewayReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("TenantGateway"),
		Recorder: mgr.GetEventRecorderFor("TenantGateway"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TenantGateway")
		return err
	}
	if err := (&gemscontroller.EnvironmentReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("Environment"),
		Recorder: mgr.GetEventRecorderFor("Environment"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Environment")
		return err
	}
	if err := (&gemscontroller.ServiceentryReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ServiceEntry"),
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	// Conditional Controllers 允许仅当crd存在时启动对应controller： https://github.com/kubernetes-sigs/controller-runtime/pull/1527
	if err := (&gemscontroller.VirtuslspaceReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Virtuslspace"),
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	if err := (&gemscontroller.PluginStatusController{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("PluginStatus"),
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	return nil
}

func setUpWebhook(mgr ctrl.Manager, setupLog logr.Logger, repo string) error {
	setupLog.Info("registering webhooks")

	// webhooks register handler
	ws := mgr.GetWebhookServer()
	c := mgr.GetClient()

	validateLogger := ctrl.Log.WithName("validate-webhook")
	validateHandler := webhooks.GetValidateHandler(&c, &validateLogger)
	ws.Register("/validate", validateHandler)

	mutateLogger := ctrl.Log.WithName("mutate-webhook")
	mutateHandler := webhooks.GetMutateHandler(&c, &mutateLogger, repo)
	ws.Register("/mutate", mutateHandler)

	labelInjectorLogger := ctrl.Log.WithName("inject-label-mutate-webhook")
	labelInjectorHandler := webhooks.GetLabelInjectorMutateHandler(&c, &labelInjectorLogger)
	ws.Register("/label-injector", labelInjectorHandler)

	return nil
}
