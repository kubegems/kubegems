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
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/rulefmt"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/set"
)

const (
	LoggingAlertRuleCMName = "kubegems-loki-rules"
	LokiRecordingRulesKey  = "kubegems-loki-recording-rules.yaml"
)

type RawLoggingAlertRule struct {
	Base *BaseAlertResource
	*corev1.ConfigMap
	*rulefmt.RuleGroups
}

type LoggingAlertRule struct {
	LogqlGenerator *LogqlGenerator `json:"logqlGenerator"`
	BaseAlertRule  `json:",inline"`
	RealTimeAlerts []*promv1.Alert `json:"realTimeAlerts,omitempty"` // 实时告警
	Origin         string          `json:"origin,omitempty"`         // 原始的prometheusrule
}

type LogqlGenerator struct {
	Duration   string            `json:"duration"`             // 时间范围
	Match      string            `json:"match"`                // 正则匹配的字符串
	LabelPairs map[string]string `json:"labelpairs,omitempty"` // 标签键值对
}

func (g *LogqlGenerator) ToLogql(namespace string) string {
	labelvalues := []string{}
	for k, v := range g.LabelPairs {
		labelvalues = append(labelvalues, fmt.Sprintf(`%s=~"%s"`, k, v))
	}
	sort.Strings(labelvalues)
	labelvalues = append(labelvalues, fmt.Sprintf(`namespace="%s"`, namespace))
	return fmt.Sprintf("sum(count_over_time({%s} |~ `%s` [%s]))without(fluentd_thread)", strings.Join(labelvalues, ", "), g.Match, g.Duration)
}

func (g *LogqlGenerator) IsEmpty() bool {
	return g == nil || g.Match == ""
}

func MutateLoggingAlert(req *LoggingAlertRule) error {
	if req.LogqlGenerator.IsEmpty() {
		if req.BaseAlertRule.Expr == "" {
			return fmt.Errorf("模板与原生promql不能同时为空")
		}
		if req.Message == "" {
			req.Message = fmt.Sprintf("%s: [集群:{{ $labels.%s }}] 触发告警, 当前值: %s", req.Name, prometheus.AlertClusterKey, prometheus.ValueAnnotationExpr)
		}
	} else {
		dur, err := model.ParseDuration(req.LogqlGenerator.Duration)
		if err != nil {
			return errors.Wrapf(err, "duration %s not valid", req.LogqlGenerator.Duration)
		}
		if time.Duration(dur).Minutes() > 10 {
			return errors.New("日志模板时长不能超过10m")
		}
		if _, err := regexp.Compile(req.LogqlGenerator.Match); err != nil {
			return errors.Wrapf(err, "match %s not valid", req.LogqlGenerator.Match)
		}
		if len(req.LogqlGenerator.LabelPairs) == 0 {
			return fmt.Errorf("labelpairs can't be null")
		}

		req.BaseAlertRule.Expr = req.LogqlGenerator.ToLogql(req.BaseAlertRule.Namespace)
		if req.Message == "" {
			req.Message = fmt.Sprintf("%s: [集群:{{ $labels.%s }}] [namespace: {{ $labels.namespace }}] ", req.Name, prometheus.AlertClusterKey)
			for label := range req.LogqlGenerator.LabelPairs {
				req.Message += fmt.Sprintf("[%s:{{ $labels.%s }}] ", label, label)
			}
			req.Message += fmt.Sprintf("日志中过去 %s 出现字符串 [%s] 次数触发告警, 当前值: %s", req.LogqlGenerator.Duration, req.LogqlGenerator.Match, prometheus.ValueAnnotationExpr)
		}
	}
	return req.BaseAlertRule.CheckAndModify()
}

func (raw *RawLoggingAlertRule) ToAlerts(hasDetail bool) (AlertRuleList[LoggingAlertRule], error) {
	receiverMap, err := raw.Base.GetAlertReceiverMap()
	if err != nil {
		return nil, err
	}
	silenceMap := raw.Base.GetSilenceMap()
	inhibitRuleMap := raw.Base.GetInhibitRuleMap()
	ret := AlertRuleList[LoggingAlertRule]{}
	for _, group := range raw.RuleGroups.Groups {
		alertrule, err := rawToLoggingAlertRule(raw.Base.AMConfig.Namespace, group)
		if err != nil {
			log.Error(err, "convert logging alert rule")
			return nil, err
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
			bts, _ := yaml.Marshal(group)
			alertrule.Origin = string(bts)
		}

		ret = append(ret, alertrule)
	}

	for k, v := range receiverMap {
		ret = append(ret, LoggingAlertRule{
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

func (raw *RawLoggingAlertRule) ModifyLoggingAlertRule(r LoggingAlertRule, act Action) error {
	alertRules, err := raw.ToAlerts(false)
	if err != nil {
		return err
	}
	alertRules, err = alertRules.modify(r, act)
	if err != nil {
		return err
	}

	groups := []rulefmt.RuleGroup{}
	for _, alertrule := range alertRules {
		if alertrule.IsExtraAlert() {
			continue
		}
		group, err := loggingAlertRuleToRaw(alertrule)
		if err != nil {
			return errors.Wrap(err, "loggingAlertRuleToRaw")
		}
		groups = append(groups, group)
	}
	raw.RuleGroups.Groups = groups
	return raw.Base.Update(alertRules.ToAlertRuleList())
}

func loggingAlertRuleToRaw(r LoggingAlertRule) (rulefmt.RuleGroup, error) {
	ret := rulefmt.RuleGroup{Name: r.Name}
	dur, err := model.ParseDuration(r.For)
	if err != nil {
		return ret, err
	}
	for _, level := range r.AlertLevels {
		rule := rulefmt.RuleNode{
			Alert: yaml.Node{Kind: yaml.ScalarNode, Value: r.Name},
			Expr:  yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%s%s%s", r.Expr, level.CompareOp, level.CompareValue)},
			For:   dur,
			Labels: map[string]string{
				prometheus.AlertNamespaceLabel: r.Namespace,
				prometheus.AlertNameLabel:      r.Name,
				prometheus.AlertFromLabel:      prometheus.AlertTypeLogging,
				prometheus.AlertScopeLabel:     getAlertScope(r.Namespace),
				prometheus.SeverityLabel:       level.Severity,
			},
			Annotations: map[string]string{
				prometheus.MessageAnnotationsKey: r.Message,
				prometheus.ValueAnnotationKey:    prometheus.ValueAnnotationExpr,
			},
		}
		if !r.LogqlGenerator.IsEmpty() {
			bts, _ := json.Marshal(r.LogqlGenerator)
			rule.Annotations[prometheus.ExprJsonAnnotationKey] = string(bts)
		}
		ret.Rules = append(ret.Rules, rule)
	}
	return ret, nil
}

func rawToLoggingAlertRule(namespace string, group rulefmt.RuleGroup) (LoggingAlertRule, error) {
	if len(group.Rules) == 0 {
		return LoggingAlertRule{}, fmt.Errorf("rule %s is null", group.Name)
	}
	ret := LoggingAlertRule{
		BaseAlertRule: BaseAlertRule{
			Namespace: namespace,
			Name:      group.Name,
			For:       group.Rules[0].For.String(),
			Message:   group.Rules[0].Annotations[prometheus.MessageAnnotationsKey],
		},
	}

	for _, rule := range group.Rules {
		if rule.Labels[prometheus.AlertNamespaceLabel] != namespace ||
			rule.Labels[prometheus.AlertNameLabel] == "" {
			return ret, fmt.Errorf("rule %s label not valid: %v", group.Name, rule.Labels)
		}
		query, op, value, hasOp := prometheus.SplitQueryExpr(rule.Expr.Value)
		if !hasOp {
			return ret, fmt.Errorf("rule %s expr %s not valid", group.Name, rule.Expr.Value)
		}

		exprJson, ok := rule.Annotations[prometheus.ExprJsonAnnotationKey]
		if ok {
			// from template
			generator := LogqlGenerator{}
			if err := json.Unmarshal([]byte(exprJson), &generator); err != nil {
				return ret, err
			}
			ret.LogqlGenerator = &generator
		}
		ret.AlertLevels = append(ret.AlertLevels, AlertLevel{
			CompareOp:    op,
			CompareValue: value,
			Severity:     rule.Labels[prometheus.SeverityLabel],
		})
		ret.BaseAlertRule.Expr = query
	}
	return ret, nil
}
