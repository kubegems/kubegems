package agentutils

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	alertmanagertypes "github.com/prometheus/alertmanager/types"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListAlertRule(ctx context.Context, cli agents.Client, opts *prometheus.MonitorOptions) ([]prometheus.AlertRule, error) {
	ruleList := monitoringv1.PrometheusRuleList{}
	configList := v1alpha1.AlertmanagerConfigList{}

	configNamespaceMap := map[string]*v1alpha1.AlertmanagerConfig{}
	silenceNamespaceMap := map[string][]alertmanagertypes.Silence{}

	ret := []prometheus.AlertRule{}
	if err := cli.List(ctx, &ruleList, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(prometheus.PrometheusRuleSelector)); err != nil {
		return nil, err
	}
	if err := cli.List(ctx, &configList, client.InNamespace(v1.NamespaceAll), client.MatchingLabels(prometheus.AlertmanagerConfigSelector)); err != nil {
		return nil, err
	}

	// 按照namespace分组
	for _, v := range configList.Items {
		if v.Name == prometheus.AlertmanagerConfigName {
			configNamespaceMap[v.Namespace] = v
		}
	}

	// 按照namespace分组
	allSilences, err := cli.Extend().ListSilences(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, silence := range allSilences {
		for _, m := range silence.Matchers {
			if m.Name == prometheus.AlertNamespaceLabel {
				silenceNamespaceMap[m.Value] = append(silenceNamespaceMap[m.Value], silence)
			}
		}
	}

	for _, rule := range ruleList.Items {
		if rule.Name == prometheus.PrometheusRuleName {
			amconfig, ok := configNamespaceMap[rule.Namespace]
			if !ok {
				log.Warnf("alertmanager config: %s not found", rule.Name)
				continue
			}
			raw := &prometheus.RawAlertResource{
				PrometheusRule:     rule,
				AlertmanagerConfig: amconfig,
				Silences:           silenceNamespaceMap[rule.Namespace],
				MonitorOptions:     opts,
			}

			alerts, err := raw.ToAlerts(false)
			if err != nil {
				return nil, err
			}
			ret = append(ret, alerts...)
		}
	}

	return ret, nil
}

// GetRawAlertResource get specified namespace's alert
func GetRawAlertResource(ctx context.Context, namespace string, cli agents.Client, opts *prometheus.MonitorOptions) (*prometheus.RawAlertResource, error) {
	amcfg, err := GetOrCreateAlertmanagerConfig(ctx, namespace, cli)
	if err != nil {
		return nil, err
	}

	promerule, err := GetOrCreatePrometheusRule(ctx, namespace, cli)
	if err != nil {
		return nil, err
	}

	silence, err := cli.Extend().ListSilences(ctx, map[string]string{
		prometheus.AlertNamespaceLabel: namespace,
	})
	if err != nil {
		return nil, err
	}

	return &prometheus.RawAlertResource{
		AlertmanagerConfig: amcfg,
		PrometheusRule:     promerule,
		Silences:           silence,
		MonitorOptions:     opts,
	}, nil
}

func GetOrCreateAlertmanagerConfig(ctx context.Context, namespace string, cli agents.Client) (*v1alpha1.AlertmanagerConfig, error) {
	aconfig := &v1alpha1.AlertmanagerConfig{}
	err := cli.Get(ctx, types.NamespacedName{Namespace: namespace, Name: prometheus.AlertmanagerConfigName}, aconfig)
	if kerrors.IsNotFound(err) {
		// 初始化
		aconfig = prometheus.GetBaseAlertmanagerConfig(namespace)
		if err := cli.Extend().CheckAlertmanagerConfig(ctx, aconfig); err != nil {
			return nil, err
		}

		if err := cli.Create(ctx, aconfig); err != nil {
			return nil, err
		}
		return aconfig, nil
	}
	return nil, err
}

func GetOrCreatePrometheusRule(ctx context.Context, namespace string, cli agents.Client) (*monitoringv1.PrometheusRule, error) {
	prule := &monitoringv1.PrometheusRule{}
	err := cli.Get(ctx, types.NamespacedName{Namespace: namespace, Name: prometheus.PrometheusRuleName}, prule)
	if kerrors.IsNotFound(err) {
		prule = prometheus.GetBasePrometheusRule(namespace)
		if err := cli.Create(ctx, prule); err != nil {
			return nil, err
		}
		return prule, err
	}
	return nil, err
}

func CommitToK8s(ctx context.Context, raw *prometheus.RawAlertResource, cli agents.Client) error {
	if err := cli.Extend().CheckAlertmanagerConfig(ctx, raw.AlertmanagerConfig); err != nil {
		return err
	}

	if err := cli.Update(ctx, raw.PrometheusRule); err != nil {
		return err
	}
	return cli.Update(ctx, raw.AlertmanagerConfig)
}
