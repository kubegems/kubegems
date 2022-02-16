package alerthandler

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/pkg/service/kubeclient"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/prometheus"
)

const (
	allNamespace = "_all"
)

var (
	defaultReceiverName = "gemcloud-default-webhook"
	defaultReceiverURL  = "http://gems-agent.gemcloud-system:8041/alert"
	defaultReceiver     = v1alpha1.Receiver{
		Name: defaultReceiverName,
		WebhookConfigs: []v1alpha1.WebhookConfig{
			{
				URL: &defaultReceiverURL,
			},
		},
	}

	nullReceiverName = "null"
	nullReceiver     = v1alpha1.Receiver{Name: nullReceiverName}
)

func (h *AlertsHandler) getRawAlertResource(ctx context.Context, cluster, namespace string) (*prometheus.RawAlertResource, error) {
	amcfg, err := h.getOrCreateAlertmanagerConfig(ctx, cluster, namespace)
	if err != nil {
		return nil, err
	}

	promerule, err := h.getOrCreatePrometheusRule(ctx, cluster, namespace)
	if err != nil {
		return nil, err
	}

	silence, err := kubeclient.GetClient().ListSilences(cluster, namespace)
	if err != nil {
		return nil, err
	}

	return &prometheus.RawAlertResource{
		AlertmanagerConfig: amcfg,
		PrometheusRule:     promerule,
		Silences:           silence,
	}, nil
}

func (h *AlertsHandler) getOrCreateAlertmanagerConfig(ctx context.Context, cluster, namespace string) (*v1alpha1.AlertmanagerConfig, error) {
	aconfig := &v1alpha1.AlertmanagerConfig{}
	err := h.Execute(ctx, cluster, func(ctx context.Context, tc agents.Client) error {
		err := tc.Get(ctx, types.NamespacedName{Namespace: namespace, Name: prometheus.AlertmanagerConfigName}, aconfig)
		if kerrors.IsNotFound(err) {
			// 初始化
			aconfig = prometheus.GetBaseAlertmanagerConfig(namespace)
			if err := kubeclient.GetClient().CheckAlertmanagerConfig(cluster, aconfig); err != nil {
				return err
			}

			return tc.Create(ctx, aconfig)
		} else {
			return err
		}
	})

	return aconfig, err
}

func (h *AlertsHandler) getOrCreatePrometheusRule(ctx context.Context, cluster, namespace string) (*monitoringv1.PrometheusRule, error) {
	prule := &monitoringv1.PrometheusRule{}
	err := h.Execute(ctx, cluster, func(ctx context.Context, tc agents.Client) error {
		err := tc.Get(ctx, types.NamespacedName{Namespace: namespace, Name: prometheus.PrometheusRuleName}, prule)
		if kerrors.IsNotFound(err) {
			prule = prometheus.GetBasePrometheusRule(namespace)
			return tc.Create(ctx, prule)
		} else {
			return err
		}
	})

	return prule, err
}

func (h *AlertsHandler) commitToK8s(ctx context.Context, cluster string, raw *prometheus.RawAlertResource) error {
	if err := kubeclient.GetClient().CheckAlertmanagerConfig(cluster, raw.AlertmanagerConfig); err != nil {
		return err
	}

	return h.Execute(ctx, cluster, func(ctx context.Context, tc agents.Client) error {
		if err := tc.Update(ctx, raw.PrometheusRule); err != nil {
			return err
		}
		return tc.Update(ctx, raw.AlertmanagerConfig)
	})
}
