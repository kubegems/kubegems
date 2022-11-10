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

package observe

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/promql/parser"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/templates"
	"kubegems.io/kubegems/pkg/utils/set"
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
	PromqlGenerator *prometheus.PromqlGenerator `json:"promqlGenerator"`

	BaseAlertRule  `json:",inline"`
	RealTimeAlerts []*promv1.Alert `json:"realTimeAlerts,omitempty"` // 实时告警
	Origin         string          `json:"origin,omitempty"`         // 原始的prometheusrule
	Source         string          `json:"source"`                   // 来自哪个prometheusrule
	TplLost        bool            `json:"tplLost"`                  // 监控模板是否丢失
}

var _ AlertRule = MonitorAlertRule{}

func MutateMonitorAlert(req *MonitorAlertRule, tplGetter templates.TplGetter) error {
	if req.Source == "" {
		return fmt.Errorf("source不能为空")
	}
	if req.PromqlGenerator.Notpl() {
		if req.BaseAlertRule.Expr == "" {
			return fmt.Errorf("模板与原生promql不能同时为空")
		}
		if req.Message == "" {
			req.Message = fmt.Sprintf("%s: [cluster:{{ $externalLabels.%s }}] trigger alert, value: %s", req.Name, prometheus.AlertClusterKey, prometheus.ValueAnnotationExpr)
		}
	} else {
		// check resource
		if err := req.PromqlGenerator.SetTpl(tplGetter); err != nil {
			return errors.Wrap(err, "set promql template")
		}

		// format message
		if req.BaseAlertRule.Message == "" {
			req.Message = fmt.Sprintf("%s: [cluster:{{ $externalLabels.%s }}] ", req.Name, prometheus.AlertClusterKey)
			for _, label := range req.PromqlGenerator.Tpl.Labels {
				req.Message += fmt.Sprintf("[%s:{{ $labels.%s }}] ", label, label)
			}

			req.Message += fmt.Sprintf("%s trigger alert, value: %s%s", req.PromqlGenerator.Tpl.RuleShowName, prometheus.ValueAnnotationExpr, req.PromqlGenerator.UnitValue.Show)
		}

		// 优先采用模板
		promql, err := req.PromqlGenerator.ToPromql(req.BaseAlertRule.Namespace)
		if err != nil {
			return errors.Wrapf(err, "template to promql: %s", req.PromqlGenerator.Tpl.Expr)
		}
		req.BaseAlertRule.Expr = promql
	}
	return req.BaseAlertRule.CheckAndModify()
}

// 默认认为namespace全部一致
func (raw *RawMonitorAlertResource) ToAlerts(hasDetail bool) (AlertRuleList[MonitorAlertRule], error) {
	receiverMap, err := raw.Base.GetAlertReceiverMap()
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
		delete(receiverMap, group.Name)

		// 是否启用
		isOpen := true
		if _, ok := silenceMap[group.Name]; ok {
			isOpen = false
		}
		alertrule.IsOpen = isOpen
		// inhibit rule
		inhitbitRule := inhibitRuleMap[group.Name]
		alertrule.InhibitLabels = set.NewSet[string]().
			Append(inhitbitRule.Equal...).
			Remove(prometheus.AlertNamespaceLabel, prometheus.AlertNameLabel).
			Slice()
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

	for k, v := range receiverMap {
		ret = append(ret, MonitorAlertRule{
			BaseAlertRule: BaseAlertRule{
				Namespace: raw.Base.AMConfig.Namespace,
				Name:      k,
				Receivers: v,
				InhibitLabels: set.NewSet[string]().
					Append(inhibitRuleMap[k].Equal...).
					Remove(prometheus.AlertNamespaceLabel, prometheus.AlertNameLabel).
					Slice(),
			},
		})
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

	groups := []monitoringv1.RuleGroup{}
	for _, alertRule := range alertRules {
		if alertRule.BaseAlertRule.IsExtraAlert() {
			continue
		}
		group, err := monitorAlertRuleToRaw(alertRule)
		if err != nil {
			return errors.Wrap(err, "monitorAlertRuleToRaw")
		}
		groups = append(groups, group)
	}

	// update PrometheusRule
	raw.PrometheusRule.Spec.Groups = groups
	return raw.Base.Update(alertRules.ToAlertRuleList())
}

func alertNamespace(ns string) string {
	if ns == prometheus.GlobalAlertNamespace {
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
				prometheus.AlertNamespaceLabel: alertRule.Namespace,
				prometheus.AlertNameLabel:      alertRule.Name,
				prometheus.AlertFromLabel:      prometheus.AlertTypeMonitor,
				prometheus.AlertScopeLabel:     getAlertScope(alertRule.Namespace),
				prometheus.SeverityLabel:       level.Severity,
			},
			Annotations: map[string]string{
				prometheus.MessageAnnotationsKey: alertRule.Message,
				prometheus.ValueAnnotationKey:    prometheus.ValueAnnotationExpr,
			},
		}
		if !alertRule.PromqlGenerator.Notpl() {
			rule.Labels[prometheus.AlertPromqlTpl] = alertRule.PromqlGenerator.TplString()
			bts, _ := json.Marshal(alertRule.PromqlGenerator)
			rule.Annotations[prometheus.ExprJsonAnnotationKey] = string(bts)
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
			Message:   group.Rules[0].Annotations[prometheus.MessageAnnotationsKey],
		},
	}
	for _, rule := range group.Rules {
		if rule.Labels[prometheus.AlertNamespaceLabel] != namespace ||
			rule.Labels[prometheus.AlertNameLabel] == "" {
			return ret, fmt.Errorf("rule %s label not valid: %v", group.Name, rule.Labels)
		}
		// from promql
		query, op, value, hasOp := prometheus.SplitQueryExpr(rule.Expr.String())
		if !hasOp {
			return ret, fmt.Errorf("rule %s expr %s not valid", group.Name, rule.Expr.String())
		}

		exprJson, ok := rule.Annotations[prometheus.ExprJsonAnnotationKey]
		if ok {
			// from template
			generator := prometheus.PromqlGenerator{}
			if err := json.Unmarshal([]byte(exprJson), &generator); err != nil {
				return ret, err
			}
			ret.PromqlGenerator = &generator
		}
		ret.BaseAlertRule.AlertLevels = append(ret.BaseAlertRule.AlertLevels, AlertLevel{
			CompareOp:    op,
			CompareValue: value,
			Severity:     rule.Labels[prometheus.SeverityLabel],
		})
		ret.BaseAlertRule.Expr = query
	}

	return ret, nil
}
