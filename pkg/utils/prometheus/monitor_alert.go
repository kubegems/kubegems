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

package prometheus

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/prometheus/promql"
	"kubegems.io/kubegems/pkg/utils/prometheus/templates"
	"kubegems.io/kubegems/pkg/utils/slice"
	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// 里面资源的namespace必须相同
type RawMonitorAlertResource struct {
	Base *BaseAlertResource
	*monitoringv1.PrometheusRule
	TplGetter templates.TplGetter
}

type MonitorAlertRule struct {
	PromqlGenerator *PromqlGenerator `json:"promqlGenerator"`

	BaseAlertRule  `json:",inline"`
	RealTimeAlerts []*promv1.Alert `json:"realTimeAlerts,omitempty"` // 实时告警
	Origin         string          `json:"origin,omitempty"`         // 原始的prometheusrule
	Source         string          `json:"source"`                   // 来自哪个prometheusrule
	TplLost        bool            `json:"tplLost"`                  // 监控模板是否丢失
}

var reg = regexp.MustCompile(`^\w+$`)

func IsValidPromqlTplName(scope, resource, rule string) error {
	if !reg.MatchString(scope) {
		return fmt.Errorf("scope not valid, must match regex: %s", reg.String())
	}
	if !reg.MatchString(resource) {
		return fmt.Errorf("resource not valid, must match regex: %s", reg.String())
	}
	if !reg.MatchString(rule) {
		return fmt.Errorf("rule not valid, must match regex: %s", reg.String())
	}
	return nil
}

type PromqlGenerator struct {
	Scope      string            `json:"scope"`                // scope
	Resource   string            `json:"resource"`             // 告警资源, eg. node、pod
	Rule       string            `json:"rule"`                 // 告警规则名, eg. cpuUsage、memoryUsagePercent
	Unit       string            `json:"unit"`                 // 单位
	LabelPairs map[string]string `json:"labelpairs,omitempty"` // 标签键值对

	UnitValue UnitValue `json:"-"`

	Tpl *templates.PromqlTpl `json:"-"`
}

func (g *PromqlGenerator) Notpl() bool {
	return g == nil || g.Resource == ""
}

func (g *PromqlGenerator) TplString() string {
	return fmt.Sprintf("%s.%s.%s", g.Scope, g.Resource, g.Rule)
}

func (g *PromqlGenerator) SetTpl(f templates.TplGetter) error {
	if err := IsValidPromqlTplName(g.Scope, g.Resource, g.Rule); err != nil {
		return err
	}
	tpl, err := f(g.Scope, g.Resource, g.Rule)
	if err != nil {
		return err
	}
	for label := range g.LabelPairs {
		if !slice.ContainStr(tpl.Labels, label) {
			return fmt.Errorf("label: %s not in tpl: %v", label, tpl.String())
		}
	}
	if g.Unit == "" {
		g.Unit = tpl.Unit
	} else if g.Unit != tpl.Unit {
		return fmt.Errorf("unit: %s not euqal with unit in template: %s", g.Unit, tpl.Unit)
	}

	unitValue, err := ParseUnit(g.Unit)
	if err != nil {
		return err
	}

	g.UnitValue = unitValue
	g.Tpl = tpl
	return nil
}

func (g *PromqlGenerator) ToPromql(namespace string) (string, error) {
	query, err := promql.New(g.Tpl.Expr)
	if err != nil {
		return "", err
	}
	if namespace != GlobalAlertNamespace && namespace != "" {
		query.AddLabelMatchers(&labels.Matcher{
			Type:  labels.MatchEqual,
			Name:  PromqlNamespaceKey,
			Value: namespace,
		})
	}

	for label, value := range g.LabelPairs {
		query.AddLabelMatchers(&labels.Matcher{
			Type:  labels.MatchRegexp,
			Name:  label,
			Value: value,
		})
	}
	return query.String(), nil
}

var _ AlertRule = MonitorAlertRule{}

func MutateMonitorAlert(req *MonitorAlertRule, f templates.TplGetter) error {
	if req.Source == "" {
		return fmt.Errorf("source不能为空")
	}
	if req.PromqlGenerator.Notpl() {
		if req.BaseAlertRule.Expr == "" {
			return fmt.Errorf("模板与原生promql不能同时为空")
		}
		if req.Message == "" {
			req.Message = fmt.Sprintf("%s: [cluster:{{ $externalLabels.%s }}] trigger alert, value: %s", req.Name, AlertClusterKey, ValueAnnotationExpr)
		}
	} else {
		// check resource
		if err := req.PromqlGenerator.SetTpl(f); err != nil {
			return errors.Wrap(err, "set promql template")
		}

		// format message
		if req.BaseAlertRule.Message == "" {
			req.Message = fmt.Sprintf("%s: [cluster:{{ $externalLabels.%s }}] ", req.Name, AlertClusterKey)
			for _, label := range req.PromqlGenerator.Tpl.Labels {
				req.Message += fmt.Sprintf("[%s:{{ $labels.%s }}] ", label, label)
			}

			req.Message += fmt.Sprintf("%s trigger alert, value: %s%s", req.PromqlGenerator.Tpl.RuleShowName, ValueAnnotationExpr, req.PromqlGenerator.UnitValue.Show)
		}

		// 优先采用模板
		promql, err := req.PromqlGenerator.ToPromql(req.BaseAlertRule.Namespace)
		if err != nil {
			return errors.Wrapf(err, "template to promql: %s", req.PromqlGenerator.Tpl.Expr)
		}
		req.BaseAlertRule.Expr = promql
	}
	return nil
}

// 默认认为namespace全部一致
func (raw *RawMonitorAlertResource) ToAlerts(hasDetail bool) (AlertRuleList[MonitorAlertRule], error) {
	receiverMap, err := raw.Base.GetReceiverMap()
	if err != nil {
		return nil, err
	}
	silenceMap := raw.Base.GetSilenceMap()
	inhibitRuleMap := raw.Base.GetInhibitRuleMap()
	ret := AlertRuleList[MonitorAlertRule]{}
	for i, group := range raw.PrometheusRule.Spec.Groups {
		// expr规则
		alertrule, err := rawToMonitorAlertRule(raw.PrometheusRule.Namespace, group)
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
		// inhibit rule
		inhitbitRule := inhibitRuleMap[group.Name]
		alertrule.InhibitLabels = slice.RemoveStr(slice.RemoveStr(inhitbitRule.Equal, AlertNamespaceLabel), AlertNameLabel)
		if len(alertrule.AlertLevels) > 1 && len(alertrule.InhibitLabels) == 0 {
			return ret, fmt.Errorf("alert rule %v inhibit label can't be null", alertrule)
		}

		if hasDetail {
			bts, _ := yaml.Marshal(raw.PrometheusRule.Spec.Groups[i])
			alertrule.Origin = string(bts)
		}
		alertrule.Source = raw.PrometheusRule.Name

		if alertrule.PromqlGenerator != nil {
			_, err = raw.TplGetter(alertrule.PromqlGenerator.Scope, alertrule.PromqlGenerator.Resource, alertrule.PromqlGenerator.Rule)
			if err != nil {
				log.Warnf("get promql tpl: %v", err)
				alertrule.TplLost = true
			}
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
		groups[i], err = monitorAlertRuleToRaw(alertRule)
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

func monitorAlertRuleToRaw(alertRule MonitorAlertRule) (monitoringv1.RuleGroup, error) {
	// 更新 PrometheusRule
	ret := monitoringv1.RuleGroup{Name: alertRule.Name}
	if _, err := parser.ParseExpr(alertRule.Expr); err != nil {
		return ret, errors.Wrapf(err, "parse expr: %s", alertRule.Expr)
	}
	for _, level := range alertRule.AlertLevels {
		rule := monitoringv1.Rule{
			Alert: alertRule.Name,
			Expr:  intstr.FromString(fmt.Sprintf("%s%s%s", alertRule.Expr, level.CompareOp, level.CompareValue)),
			For:   alertRule.For,
			Labels: map[string]string{
				AlertNamespaceLabel: alertRule.Namespace,
				AlertNameLabel:      alertRule.Name,
				AlertFromLabel:      AlertTypeMonitor,
				AlertScopeLabel:     getAlertScope(alertRule.Namespace),
				SeverityLabel:       level.Severity,
			},
			Annotations: map[string]string{
				MessageAnnotationsKey: alertRule.Message,
				valueAnnotationKey:    ValueAnnotationExpr,
			},
		}
		if !alertRule.PromqlGenerator.Notpl() {
			rule.Labels[AlertPromqlTpl] = alertRule.PromqlGenerator.TplString()
			bts, _ := json.Marshal(alertRule.PromqlGenerator)
			rule.Annotations[ExprJsonAnnotationKey] = string(bts)
		}
		ret.Rules = append(ret.Rules, rule)
	}
	return ret, nil
}

// TODO: unit test
func rawToMonitorAlertRule(namespace string, group monitoringv1.RuleGroup) (MonitorAlertRule, error) {
	if len(group.Rules) == 0 {
		return MonitorAlertRule{}, fmt.Errorf("rule %s is null", group.Name)
	}
	ret := MonitorAlertRule{
		BaseAlertRule: BaseAlertRule{
			Namespace: namespace,
			Name:      group.Name,
			For:       group.Rules[0].For,
			Message:   group.Rules[0].Annotations[MessageAnnotationsKey],
		},
	}
	for _, rule := range group.Rules {
		if rule.Labels[AlertNamespaceLabel] != namespace ||
			rule.Labels[AlertNameLabel] == "" {
			return ret, fmt.Errorf("rule %s label not valid: %v", group.Name, rule.Labels)
		}
		// from promql
		query, op, value, hasOp := SplitQueryExpr(rule.Expr.String())
		if !hasOp {
			return ret, fmt.Errorf("rule %s expr %s not valid", group.Name, rule.Expr.String())
		}

		exprJson, ok := rule.Annotations[ExprJsonAnnotationKey]
		if ok {
			// from template
			generator := PromqlGenerator{}
			if err := json.Unmarshal([]byte(exprJson), &generator); err != nil {
				return ret, err
			}
			ret.PromqlGenerator = &generator
		}
		ret.BaseAlertRule.AlertLevels = append(ret.BaseAlertRule.AlertLevels, AlertLevel{
			CompareOp:    op,
			CompareValue: value,
			Severity:     rule.Labels[SeverityLabel],
		})
		ret.BaseAlertRule.Expr = query
	}

	return ret, nil
}

func RealTimeAlertKey(namespace, name string) string {
	return fmt.Sprintf(alertRuleKeyFormat, namespace, name)
}
