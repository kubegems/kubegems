package prometheus

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/apis/gems"
)

type Action int

const (
	Add Action = iota
	Update
	Delete
)

const (
	// prometheusrule and alertmanagerconfigname
	MonitorAlertCRDName = "kubegems-default-monitor-alert-rule"
	LoggingAlertCRDName = "kubegems-default-logging-alert-rule"
)

type RealTimeAlertRule struct {
	Name string `json:"name"`
	// Query          string         `json:"query"`
	// Duration       float64        `json:"duration"`
	// Labels         model.LabelSet `json:"labels"`
	// Annotations    model.LabelSet `json:"annotations"`
	Alerts []*v1.Alert `json:"alerts"`
	// Health         v1.RuleHealth  `json:"health"`
	// LastError      string         `json:"lastError,omitempty"`
	// EvaluationTime float64        `json:"evaluationTime"`
	// LastEvaluation time.Time      `json:"lastEvaluation"`
	State string `json:"state"`
	// Type           v1.RuleType    `json:"type"`
}

func (r *RealTimeAlertRule) Len() int      { return len(r.Alerts) }
func (r *RealTimeAlertRule) Swap(i, j int) { r.Alerts[i], r.Alerts[j] = r.Alerts[j], r.Alerts[i] }
func (r *RealTimeAlertRule) Less(i, j int) bool {
	return r.Alerts[i].ActiveAt.After(r.Alerts[j].ActiveAt)
} // 倒排

func GetBaseAlertmanagerConfig(namespace, name string) *v1alpha1.AlertmanagerConfig {
	return &v1alpha1.AlertmanagerConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.AlertmanagerConfigKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				gems.LabelAlertmanagerConfigName: name,
				gems.LabelAlertmanagerConfigType: func() string {
					if name == LoggingAlertCRDName {
						return AlertTypeLogging
					} else {
						return AlertTypeMonitor
					}
				}(),
			},
		},
		Spec: v1alpha1.AlertmanagerConfigSpec{
			Route: &v1alpha1.Route{
				GroupBy:       []string{AlertNamespaceLabel, AlertNameLabel},
				GroupWait:     "30s",
				GroupInterval: "30s",
				Continue:      false,
				Receiver:      NullReceiverName, // 默认发给空接收器，避免defaultReceiver收到不该收到的alert
			},
			Receivers:    []v1alpha1.Receiver{NullReceiver, DefaultReceiver},
			InhibitRules: []v1alpha1.InhibitRule{},
		},
	}
}

func GetBasePrometheusRule(namespace, name string) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       monitoringv1.PrometheusRuleKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				gems.LabelPrometheusRuleName: name,
				gems.LabelPrometheusRuleType: AlertTypeMonitor,
			},
		},
		Spec: monitoringv1.PrometheusRuleSpec{},
	}
}
