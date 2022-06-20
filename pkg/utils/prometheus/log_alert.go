package prometheus

import (
	"fmt"
	"regexp"

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
	LoggingAlertRuleCMName  = "loki-alerting-rules"
	LoggingRecordingRuleKey = "loki-alerting-rules.yaml" // loki helm 部署 recording rule就是这个名字。。
)

type RawLoggingAlertRule struct {
	Base *BaseAlertResource
	*corev1.ConfigMap
	*rulefmt.RuleGroups
}

type LoggingAlertRule struct {
	BaseAlertRule  `json:",inline"`
	RealTimeAlerts []*promv1.Alert `json:"realTimeAlerts,omitempty"` // 实时告警
	Origin         string          `json:"origin,omitempty"`         // 原始的prometheusrule
}

func (r *LoggingAlertRule) CheckAndModify(opts *MonitorOptions) error {
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
		ret.Expr = query
		ret.AlertLevels = append(ret.AlertLevels, AlertLevel{
			CompareOp:    op,
			CompareValue: value,
			Severity:     rule.Labels[SeverityLabel],
		})
	}
	return ret, nil
}

func loggingAlertRuleToRaw(r LoggingAlertRule) (rulefmt.RuleGroup, error) {
	ret := rulefmt.RuleGroup{Name: r.Name}
	dur, err := model.ParseDuration(r.For)
	if err != nil {
		return ret, err
	}
	for _, level := range r.AlertLevels {
		ret.Rules = append(ret.Rules, rulefmt.RuleNode{
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
		})
	}
	return ret, nil
}
