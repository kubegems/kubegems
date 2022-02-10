package prometheus

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/prometheus/promql"
)

type Action int

const (
	Add Action = iota
	Update
	Delete
)

const (
	AlertmanagerConfigName = "gemcloud"
	PrometheusRuleName     = "gemcloud"
)

var (
	AlertmanagerConfigSelector = map[string]string{
		"alertmanagerConfig": "gemcloud",
	}
	PrometheusRuleSelector = map[string]string{
		"prometheusRule": "gemcloud",
	}

	// 单位表
	UnitValueMap = map[string]UnitValue{
		"percent": defaultUnitValue,

		"core":  defaultUnitValue,
		"mcore": {Op: "*", Value: "1000"},

		"b":  defaultUnitValue,
		"kb": {Op: "/", Value: "1024"},
		"mb": {Op: "/", Value: "(1024 * 1024)"},
		"gb": {Op: "/", Value: "(1024 * 1024 * 1024)"},
		"tb": {Op: "/", Value: "(1024 * 1024 * 1024 * 1024)"},

		"bps":  defaultUnitValue,
		"kbps": {Op: "/", Value: "1024"},
		"mbps": {Op: "/", Value: "(1024 * 1024)"},

		"ops":   defaultUnitValue,
		"count": defaultUnitValue,
		"times": defaultUnitValue,

		"us": {Op: "*", Value: "(1000 * 1000)"},
		"ms": {Op: "*", Value: "1000"},
		"s":  defaultUnitValue,
		"m":  {Op: "/", Value: "60"},
		"h":  {Op: "/", Value: "(60 * 60)"},
		"d":  {Op: "/", Value: "(24 * 60 * 60)"},
		"w":  {Op: "/", Value: "(7 * 24 * 60 * 60)"},
	}

	defaultUnitValue = UnitValue{
		Op:    "*",
		Value: "1",
	}
)

type UnitValue struct {
	Op    string
	Value string
}

type BaseQueryParams struct {
	Resource string `json:"resource"` // 告警资源, eg. node、pod
	Rule     string `json:"rule"`     // 告警规则名, eg. cpuUsage、memoryUsagePercent
	Unit     string `json:"unit"`     // 单位

	LabelPairs map[string]string `json:"labelpairs,omitempty"` // 标签键值对
}

type CompareQueryParams struct {
	BaseQueryParams `json:",inline"`
	CompareOp       string `json:"compareOp"`
	CompareValue    string `json:"compareValue"`
}

type RuleContext struct {
	ResourceDetail ResourceDetail
	RuleDetail     RuleDetail
}

// 查询规则上下文
func (params *BaseQueryParams) FindRuleContext(cfg GemsMetricConfig) (RuleContext, error) {
	ctx := RuleContext{}
	resourceDetail, ok := cfg.Resources[params.Resource]
	if !ok {
		return ctx, fmt.Errorf("invalid resource: %s", params.Resource)
	}

	ruleDetail, ok := resourceDetail.Rules[params.Rule]
	if !ok {
		return ctx, fmt.Errorf("rule %s not in resource %s", params.Rule, params.Resource)
	}

	if !(ruleDetail.Units == nil && params.Unit == "") {
		if !utils.ContainStr(ruleDetail.Units, params.Unit) {
			return ctx, fmt.Errorf("invalid unit: %s in ruledetail: %v", params.Unit, ruleDetail)
		}
	}

	for label := range params.LabelPairs {
		if !utils.ContainStr(ruleDetail.Labels, label) {
			return ctx, fmt.Errorf("invalid label: %s in ruledetail: %v", label, ruleDetail)
		}
	}
	ctx.ResourceDetail = resourceDetail
	ctx.RuleDetail = ruleDetail
	return ctx, nil
}

func (params *CompareQueryParams) ConstructPromql(namespace string) (string, error) {
	ruleCtx, err := params.FindRuleContext(GetGemsMetricConfig(true))
	if err != nil {
		return "", fmt.Errorf("constructPromql params: %v, err: %w", params, err)
	}
	query := promql.New(ruleCtx.RuleDetail.Expr)
	if namespace != GlobalAlertNamespace {
		query.AddSelector(PromqlNamespaceKey, promql.LabelEqual, namespace)
	}

	for label, value := range params.LabelPairs {
		query.AddSelector(label, promql.LabelRegex, value)
	}
	return query.
		Arithmetic(promql.BinaryArithmeticOperators(UnitValueMap[params.Unit].Op), UnitValueMap[params.Unit].Value).
		Compare(promql.ComparisonOperator(params.CompareOp), params.CompareValue).
		ToPromql(), nil
}

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

func GetBaseAlertmanagerConfig(namespace string) *v1alpha1.AlertmanagerConfig {
	return &v1alpha1.AlertmanagerConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.AlertmanagerConfigKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      AlertmanagerConfigName,
			Namespace: namespace,
			Labels:    AlertmanagerConfigSelector,
		},
		Spec: v1alpha1.AlertmanagerConfigSpec{
			Route: &v1alpha1.Route{
				GroupBy:       []string{AlertNamespaceLabel, AlertNameLabel},
				GroupWait:     "30s",
				GroupInterval: "30s",
				Continue:      true,
				Receiver:      NullReceiverName, // 默认发给空接收器，避免defaultReceiver收到不该收到的alert
			},
			Receivers:    []v1alpha1.Receiver{NullReceiver, DefaultReceiver},
			InhibitRules: []v1alpha1.InhibitRule{},
		},
	}
}

func GetBasePrometheusRule(namespace string) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       monitoringv1.PrometheusRuleKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      PrometheusRuleName,
			Namespace: namespace,
			Labels:    PrometheusRuleSelector,
		},
		Spec: monitoringv1.PrometheusRuleSpec{},
	}
}
