package prometheus

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
	"kubegems.io/pkg/log"
)

// 里面资源的namespace必须相同
type RawMonitorAlertResource struct {
	Base *BaseAlertResource
	*monitoringv1.PrometheusRule
	*MonitorOptions
}

type MonitorAlertRule struct {
	BaseQueryParams `json:",inline"`

	*BaseAlertRule `json:",inline"`

	RealTimeAlerts []*promv1.Alert `json:"realTimeAlerts,omitempty"` // 实时告警

	Promql string `json:"promql"` // 仅用于告知前端展示
	// 相关配置
	RuleContext `json:"-"`
	Origin      string `json:"origin,omitempty"` // 原始的prometheusrule
}

var _ AlertRule = MonitorAlertRule{}

// TODO: unit test
func (r *MonitorAlertRule) CheckAndModify(opts *MonitorOptions) error {
	// check resource
	res, ok := opts.Resources[r.Resource]
	if !ok {
		return fmt.Errorf("invalid resource: %s", r.Resource)
	}

	// check rule
	ruleCtx, err := r.FindRuleContext(opts)
	if err != nil {
		return err
	}
	r.RuleContext = ruleCtx

	// format message
	if r.Message == "" {
		r.Message = fmt.Sprintf("%s: [集群:{{ $externalLabels.%s }}] ", r.Name, AlertClusterKey)
		for _, label := range r.RuleDetail.Labels {
			r.Message += fmt.Sprintf("[%s:{{ $labels.%s }}] ", label, label)
		}

		r.Message += fmt.Sprintf("%s%s 触发告警, 当前值: %s%s", res.ShowName, r.RuleDetail.ShowName, valueAnnotationExpr, opts.Units[r.Unit])
	}

	return r.BaseAlertRule.checkAndModify(opts)
}

// 默认认为namespace全部一致
func (raw *RawMonitorAlertResource) ToAlerts(hasDetail bool) (AlertRuleList[MonitorAlertRule], error) {
	receiverMap, err := raw.Base.GetReceiverMap()
	if err != nil {
		return nil, err
	}
	silenceMap := raw.Base.GetSilenceMap()

	ret := AlertRuleList[MonitorAlertRule]{}
	for i, group := range raw.PrometheusRule.Spec.Groups {
		// expr规则
		alertrule, err := rawToMonitorAlertRule(raw.PrometheusRule.Namespace, group, raw.MonitorOptions)
		if err != nil {
			log.Error(err, "convert prometheus rule")
			return nil, err
		}
		// 接收器
		alertrule.Receivers = receiverMap[group.Name]

		// 是否启用
		isOpen := true
		if _, ok := silenceMap[group.Name]; ok {
			isOpen = false
		}
		alertrule.IsOpen = isOpen
		if hasDetail {
			bts, _ := yaml.Marshal(raw.PrometheusRule.Spec.Groups[i])
			alertrule.Origin = string(bts)
		}
		ret = append(ret, alertrule)
	}

	return ret, nil
}

// 所有alertrule都是一个namespace
func (raw *RawMonitorAlertResource) ModifyAlertRule(newAlertRule MonitorAlertRule, act Action) error {
	alertRules, err := raw.ToAlerts(false)
	if err != nil {
		return err
	}

	alertRules, err = alertRules.modify(newAlertRule, act)
	if err != nil {
		return err
	}

	groups := make([]monitoringv1.RuleGroup, len(alertRules))
	for i, alertRule := range alertRules {
		groups[i], err = monitorAlertRuleToRaw(alertRule, raw.MonitorOptions)
		if err != nil {
			return errors.Wrap(err, "monitorAlertRuleToRaw")
		}
		// 监控告警规则模板有labels，覆盖用户配置
		alertRules[i].InhibitLabels = alertRules[i].RuleDetail.Labels
	}

	// update PrometheusRule
	raw.PrometheusRule.Spec.Groups = groups
	// update AlertmanagerConfig routes
	raw.Base.UpdateRoutes(alertRules.ToAlertRuleList())
	// add null receivers
	raw.Base.AddNullReceivers()
	// update AlertmanagerConfig inhibit rules
	return raw.Base.UpdateInhibitRules(alertRules.ToAlertRuleList())
}

func monitorAlertRuleToRaw(alertRule MonitorAlertRule, opts *MonitorOptions) (monitoringv1.RuleGroup, error) {
	// 更新 PrometheusRule
	ret := monitoringv1.RuleGroup{Name: alertRule.Name}
	for _, level := range alertRule.AlertLevels {
		expr := CompareQueryParams{
			BaseQueryParams: alertRule.BaseQueryParams,
			CompareOp:       level.CompareOp,
			CompareValue:    level.CompareValue,
		}
		bts, _ := json.Marshal(expr)

		promql, err := expr.ConstructPromql(alertRule.Namespace, opts)
		if err != nil {
			return ret, err
		}
		ret.Rules = append(ret.Rules, monitoringv1.Rule{
			Alert: alertRule.Name,
			Expr:  intstr.FromString(promql),
			For:   alertRule.For,
			Labels: map[string]string{
				AlertNamespaceLabel: alertRule.Namespace,
				AlertNameLabel:      alertRule.Name,
				AlertResourceLabel:  alertRule.Resource,
				AlertRuleLabel:      alertRule.Rule,
				AlertScopeLabel:     getAlertScope(alertRule.Namespace),
				SeverityLabel:       level.Severity,
			},
			Annotations: map[string]string{
				messageAnnotationsKey: alertRule.Message,
				valueAnnotationKey:    valueAnnotationExpr,
				exprJsonAnnotationKey: string(bts),
			},
		})
	}
	return ret, nil
}

// TODO: unit test
func rawToMonitorAlertRule(namespace string, group monitoringv1.RuleGroup, opts *MonitorOptions) (MonitorAlertRule, error) {
	if len(group.Rules) == 0 {
		return MonitorAlertRule{}, fmt.Errorf("rule %s is null", group.Name)
	}
	ret := MonitorAlertRule{
		BaseAlertRule: &BaseAlertRule{
			Namespace: namespace,
			Name:      group.Name,
			For:       group.Rules[0].For,
			Message:   group.Rules[0].Annotations[messageAnnotationsKey],
		},
	}
	tmpexpr := CompareQueryParams{}
	for _, rule := range group.Rules {
		if rule.Labels[AlertNamespaceLabel] != namespace ||
			rule.Labels[AlertNameLabel] == "" {
			return ret, fmt.Errorf("rule %s label not valid: %v", group.Name, rule.Labels)
		}

		exprJson := rule.Annotations[exprJsonAnnotationKey]
		if exprJson == "" {
			return ret, fmt.Errorf("rule annotation not contains expr json key: %v", rule)
		}

		expr := CompareQueryParams{}
		if err := json.Unmarshal([]byte(exprJson), &expr); err != nil {
			return ret, err
		}

		ret.BaseQueryParams = expr.BaseQueryParams
		ret.AlertLevels = append(ret.AlertLevels, AlertLevel{
			CompareOp:    expr.CompareOp,
			CompareValue: expr.CompareValue,
			Severity:     rule.Labels[SeverityLabel],
		})

		tmpexpr.BaseQueryParams = expr.BaseQueryParams
	}

	// 填入ruleCtx
	ruleCtx, err := ret.FindRuleContext(opts)
	if err != nil {
		return ret, err
	}
	ret.RuleContext = ruleCtx

	// 填入promql
	promql, err := tmpexpr.ConstructPromql(namespace, opts)
	if err != nil {
		return ret, nil
	}
	ret.Promql = promql

	return ret, err
}

func RealTimeAlertKey(namespace, name string) string {
	return fmt.Sprintf(alertRuleKeyFormat, namespace, name)
}
