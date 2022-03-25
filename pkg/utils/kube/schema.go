package kube

import (
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	loggingv1beta1 "github.com/banzaicloud/logging-operator/pkg/sdk/api/v1beta1"
	csiv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	istiopkgv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	applicationv1beta1 "kubegems.io/pkg/apis/application/v1beta1"
	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
	pluginv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
)

func AddToschema(schema *runtime.Scheme) {
	_ = clientgoscheme.AddToScheme(schema)
	_ = metricsv1beta1.AddToScheme(schema)
	_ = monitoringv1.AddToScheme(schema)
	_ = monitoringv1alpha1.AddToScheme(schema)
	_ = gemsv1beta1.AddToScheme(schema)
	_ = argocdv1alpha1.AddToScheme(schema)
	_ = csiv1.AddToScheme(schema)
	_ = rolloutsv1alpha1.AddToScheme(schema)
	_ = istiov1beta1.AddToScheme(schema)
	_ = istiopkgv1alpha1.SchemeBuilder.AddToScheme(schema)
	_ = applicationv1beta1.AddToScheme(schema)
	_ = loggingv1beta1.AddToScheme(schema)
	_ = pluginv1beta1.AddToScheme(schema)
}

func GetScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	AddToschema(scheme)
	return scheme
}
