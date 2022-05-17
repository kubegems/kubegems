package prometheus

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/promql/parser"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// 里面资源的namespace必须相同
type RawMonitorAlertResource struct {
	Base *BaseAlertResource
	*monitoringv1.PrometheusRule
	*MonitorOptions
}

type MonitorAlertRule struct {
	*PromqlGenerator `json:"promqlGenerator"`

	BaseAlertRule  `json:",inline"`
	RealTimeAlerts []*promv1.Alert `json:"realTimeAlerts,omitempty"` // 实时告警
	Origin         string          `json:"origin,omitempty"`         // 原始的prometheusrule
}

var _ AlertRule = MonitorAlertRule{}

// TODO: unit test
func (r *MonitorAlertRule) CheckAndModify(opts *MonitorOptions) error {
	if r.PromqlGenerator == nil && r.BaseAlertRule.Expr == "" {
		return fmt.Errorf("模板与原生promql不能同时为空")
	}
	if r.PromqlGenerator != nil {
		if r.BaseAlertRule.Expr != "" {
			return fmt.Errorf("模板与原生promql只能指定一种")
		}
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
		r.PromqlGenerator.RuleContext = ruleCtx

		// format message
		if r.BaseAlertRule.Message == "" {
			r.Message = fmt.Sprintf("%s: [集群:{{ $externalLabels.%s }}] ", r.Name, AlertClusterKey)
			for _, label := range r.RuleDetail.Labels {
				r.Message += fmt.Sprintf("[%s:{{ $labels.%s }}] ", label, label)
			}

			r.Message += fmt.Sprintf("%s%s 触发告警, 当前值: %s%s", res.ShowName, r.RuleDetail.ShowName, valueAnnotationExpr, opts.Units[r.Unit])
		}

		// 优先采用模板
		promql, err := r.PromqlGenerator.ConstructPromql(r.BaseAlertRule.Namespace, opts)
		if err != nil {
			return errors.Wrap(err, "ConstructPromql")
		}
		r.BaseAlertRule.Expr = promql
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
			return nil, errors.Wrap(err, "rawToMonitorAlertRule")
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

func alertNamespace(ns string) string {
	if ns == GlobalAlertNamespace {
		return ""
	}
	return ns
}

func monitorAlertRuleToRaw(alertRule MonitorAlertRule, opts *MonitorOptions) (monitoringv1.RuleGroup, error) {
	// 更新 PrometheusRule
	ret := monitoringv1.RuleGroup{Name: alertRule.Name}
	if _, err := parser.ParseExpr(alertRule.Expr); err != nil {
		return ret, err
	}
	for _, level := range alertRule.AlertLevels {
		if alertRule.PromqlGenerator == nil {
			ret.Rules = append(ret.Rules, monitoringv1.Rule{
				Alert: alertRule.Name,
				Expr:  intstr.FromString(fmt.Sprintf("%s%s%s", alertRule.Expr, level.CompareOp, level.CompareValue)),
				For:   alertRule.For,
				Labels: map[string]string{
					AlertNamespaceLabel: alertRule.Namespace,
					AlertNameLabel:      alertRule.Name,
					AlertFromLabel:      AlertFromMonitor,
					AlertScopeLabel:     getAlertScope(alertRule.Namespace),
					SeverityLabel:       level.Severity,
				},
				Annotations: map[string]string{
					messageAnnotationsKey: alertRule.Message,
					valueAnnotationKey:    valueAnnotationExpr,
				},
			})
		} else {
			expr := PromqlGenerator{
				BaseQueryParams: alertRule.BaseQueryParams,
				CompareOp:       level.CompareOp,
				CompareValue:    level.CompareValue,
			}
			bts, _ := json.Marshal(expr)

			promql, err := expr.ConstructPromql(alertNamespace(alertRule.Namespace), opts)
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
					AlertFromLabel:      AlertFromMonitor,
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
	}
	return ret, nil
}

// TODO: unit test
func rawToMonitorAlertRule(namespace string, group monitoringv1.RuleGroup, opts *MonitorOptions) (MonitorAlertRule, error) {
	if len(group.Rules) == 0 {
		return MonitorAlertRule{}, fmt.Errorf("rule %s is null", group.Name)
	}
	ret := MonitorAlertRule{
		BaseAlertRule: BaseAlertRule{
			Namespace: namespace,
			Name:      group.Name,
			For:       group.Rules[0].For,
			Message:   group.Rules[0].Annotations[messageAnnotationsKey],
		},
	}
	for _, rule := range group.Rules {
		if rule.Labels[AlertNamespaceLabel] != namespace ||
			rule.Labels[AlertNameLabel] == "" {
			return ret, fmt.Errorf("rule %s label not valid: %v", group.Name, rule.Labels)
		}

		exprJson, ok := rule.Annotations[exprJsonAnnotationKey]
		if ok {
			// from template
			generator := PromqlGenerator{}
			if err := json.Unmarshal([]byte(exprJson), &generator); err != nil {
				return ret, err
			}
			ret.BaseAlertRule.AlertLevels = append(ret.BaseAlertRule.AlertLevels, AlertLevel{
				CompareOp:    generator.CompareOp,
				CompareValue: generator.CompareValue,
				Severity:     rule.Labels[SeverityLabel],
			})
			// 填入ruleCtx
			ruleCtx, err := generator.BaseQueryParams.FindRuleContext(opts)
			if err != nil {
				return ret, err
			}
			generator.RuleContext = ruleCtx

			// 解析 expr
			generator.CompareOp = ""
			generator.CompareValue = ""
			expr, err := generator.ConstructPromql(alertNamespace(namespace), opts)
			if err != nil {
				return ret, errors.Wrap(err, "ConstructPromql")
			}

			ret.BaseAlertRule.Expr = expr
			ret.PromqlGenerator = &generator
		} else {
			// from promql
			query, op, value, hasOp := SplitQueryExpr(rule.Expr.String())
			if !hasOp {
				return ret, fmt.Errorf("rule %s expr %s not valid", group.Name, rule.Expr.String())
			}
			ret.BaseAlertRule.Expr = query
			ret.BaseAlertRule.AlertLevels = append(ret.BaseAlertRule.AlertLevels, AlertLevel{
				CompareOp:    op,
				CompareValue: value,
				Severity:     rule.Labels[SeverityLabel],
			})
		}
	}

	return ret, nil
}

func RealTimeAlertKey(namespace, name string) string {
	return fmt.Sprintf(alertRuleKeyFormat, namespace, name)
}
