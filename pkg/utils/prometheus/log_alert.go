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
	"kubegems.io/kubegems/pkg/utils/slice"
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
	return fmt.Sprintf(`sum(count_over_time({%s} |~ "%s" [%s]))without(fluentd_thread)`, strings.Join(labelvalues, ", "), g.Match, g.Duration)
}

func (g *LogqlGenerator) IsEmpty() bool {
	return g == nil || g.Match == ""
}

func (r *LoggingAlertRule) CheckAndModify(opts *MonitorOptions) error {
	if r.LogqlGenerator.IsEmpty() {
		if r.BaseAlertRule.Expr == "" {
			return fmt.Errorf("模板与原生promql不能同时为空")
		}
		if r.Message == "" {
			r.Message = fmt.Sprintf("%s: [集群:{{ $labels.%s }}] 触发告警, 当前值: %s", r.Name, AlertClusterKey, valueAnnotationExpr)
		}
	} else {
		dur, err := model.ParseDuration(r.LogqlGenerator.Duration)
		if err != nil {
			return errors.Wrapf(err, "duration %s not valid", r.LogqlGenerator.Duration)
		}
		if time.Duration(dur).Minutes() > 10 {
			return errors.New("日志模板时长不能超过10m")
		}
		if _, err := regexp.Compile(r.LogqlGenerator.Match); err != nil {
			return errors.Wrapf(err, "match %s not valid", r.LogqlGenerator.Match)
		}
		if len(r.LogqlGenerator.LabelPairs) == 0 {
			return fmt.Errorf("labelpairs can't be null")
		}

		r.BaseAlertRule.Expr = r.LogqlGenerator.ToLogql(r.BaseAlertRule.Namespace)
		if r.Message == "" {
			r.Message = fmt.Sprintf("%s: [集群:{{ $labels.%s }}] [namespace: {{ $labels.namespace }}] ", r.Name, AlertClusterKey)
			for label := range r.LogqlGenerator.LabelPairs {
				r.Message += fmt.Sprintf("[%s:{{ $labels.%s }}] ", label, label)
			}
			r.Message += fmt.Sprintf("日志中过去 %s 出现字符串 [%s] 次数触发告警, 当前值: %s", r.LogqlGenerator.Duration, r.LogqlGenerator.Match, valueAnnotationExpr)
		}
	}
	return r.BaseAlertRule.checkAndModify(opts)
}

var logqlReg = regexp.MustCompile("(.*)(<|<=|==|!=|>|>=)(.*)")

func SplitQueryExpr(logql string) (query, op, value string, hasOp bool) {
	substrs := logqlReg.FindStringSubmatch(logql)
	if len(substrs) == 4 {
		query = substrs[1]
		op = substrs[2]
		value = substrs[3]
		hasOp = true
	} else {
		query = logql
	}
	return
}

func (raw *RawLoggingAlertRule) ToAlerts(hasDetail bool) (AlertRuleList[LoggingAlertRule], error) {
	receiverMap, err := raw.Base.GetReceiverMap()
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
			bts, _ := yaml.Marshal(group)
			alertrule.Origin = string(bts)
		}

		ret = append(ret, alertrule)
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

	groups := make([]rulefmt.RuleGroup, len(alertRules))
	for i, alertrule := range alertRules {
		groups[i], err = loggingAlertRuleToRaw(alertrule)
		if err != nil {
			return errors.Wrap(err, "loggingAlertRuleToRaw")
		}
	}
	raw.RuleGroups.Groups = groups
	// update AlertmanagerConfig routes
	raw.Base.UpdateRoutes(alertRules.ToAlertRuleList())
	// add null receivers
	raw.Base.AddNullReceivers()
	// update AlertmanagerConfig inhibit rules
	return raw.Base.UpdateInhibitRules(alertRules.ToAlertRuleList())
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
				AlertNamespaceLabel: r.Namespace,
				AlertNameLabel:      r.Name,
				AlertFromLabel:      AlertTypeLogging,
				AlertScopeLabel:     getAlertScope(r.Namespace),
				SeverityLabel:       level.Severity,
			},
			Annotations: map[string]string{
				messageAnnotationsKey: r.Message,
				valueAnnotationKey:    valueAnnotationExpr,
			},
		}
		if !r.LogqlGenerator.IsEmpty() {
			bts, _ := json.Marshal(r.LogqlGenerator)
			rule.Annotations[exprJsonAnnotationKey] = string(bts)
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
			Message:   group.Rules[0].Annotations[messageAnnotationsKey],
		},
	}

	for _, rule := range group.Rules {
		if rule.Labels[AlertNamespaceLabel] != namespace ||
			rule.Labels[AlertNameLabel] == "" {
			return ret, fmt.Errorf("rule %s label not valid: %v", group.Name, rule.Labels)
		}
		query, op, value, hasOp := SplitQueryExpr(rule.Expr.Value)
		if !hasOp {
			return ret, fmt.Errorf("rule %s expr %s not valid", group.Name, rule.Expr.Value)
		}

		exprJson, ok := rule.Annotations[exprJsonAnnotationKey]
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
			Severity:     rule.Labels[SeverityLabel],
		})
		ret.BaseAlertRule.Expr = query
	}
	return ret, nil
}
