package prometheus

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	alertmanagertypes "github.com/prometheus/alertmanager/types"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"kubegems.io/pkg/utils/slice"
)

const (
	AlertNamespaceLabel = "gems_namespace"
	AlertNameLabel      = "gems_alertname"
	// 用于从告警中获取告警资源
	AlertResourceLabel = "gems_alert_resource"
	AlertRuleLabel     = "gems_alert_rule"

	SeverityLabel    = "severity"
	SeverityError    = "error"    // 错误
	SeverityCritical = "critical" // 严重

	exprJsonAnnotationKey = "gems_expr_json"
	MessageAnnotationsKey = "message"
	ValueAnnotationKey    = "value"

	alertRuleKeyFormat = "gems-%s-%s"
	AlertClusterKey    = "cluster"

	// 告警消息发送范围
	AlertScopeLabel  = "gems_alert_scope"
	ScopeSystemAdmin = "system-admin" // 系统管理员
	ScopeSystemUser  = "system-user"  // 所有用户
	ScopeNormal      = "normal"       // 普通租户用户

	SilenceCommentForBlackListPrefix = "fingerprint-"
	SilenceCommentForAlertrulePrefix = "silence for"
)

type AlertLevel struct {
	CompareOp    string `json:"compareOp"`
	CompareValue string `json:"compareValue"` // 支持表达式, eg. 24 * 60
	Severity     string `json:"severity"`     // error, critical
}

type AlertReceiver struct {
	Name     string `json:"name"`
	Interval string `json:"interval"` // 分组间隔
}

type BaseAlertRule struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`

	For     string `json:"for"`     // 持续时间, eg. 10s, 1m, 1h
	Message string `json:"message"` // 告警消息，若为空后端自动填充

	InhibitLabels []string        // 如果有多个告警级别，需要配置告警抑制的labels
	AlertLevels   []AlertLevel    `json:"alertLevels"` // 告警级别
	Receivers     []AlertReceiver `json:"receivers"`   // 接收器

	IsOpen bool   `json:"isOpen"` // 是否启用
	State  string `json:"state"`  // 状态
}

type AlertRule interface {
	GetNamespace() string
	GetName() string
	GetInhibitLabels() []string
	GetAlertLevels() []AlertLevel
	GetReceivers() []AlertReceiver
}

type AlertRuleList[T AlertRule] []T

func (l AlertRuleList[T]) ToAlertRuleList() []AlertRule {
	ret := make([]AlertRule, len(l))
	for i, v := range l {
		ret[i] = v
	}
	return ret
}

func (l AlertRuleList[T]) modify(newAlertRule T, act Action) (AlertRuleList[T], error) {
	index := -1
	for i := range l {
		if l[i].GetName() == newAlertRule.GetName() {
			index = i
			break
		}
	}

	switch act {
	case Add:
		if index != -1 { // found
			return l, fmt.Errorf("告警规则 %s 已存在！", newAlertRule.GetName())
		}
		l = append(l, newAlertRule)
	case Update:
		if index == -1 { // not found
			return l, fmt.Errorf("告警规则 %s 不存在！", newAlertRule.GetName())
		}
		l[index] = newAlertRule
	case Delete:
		if index == -1 { // not found
			return l, fmt.Errorf("告警规则 %s 不存在！", newAlertRule.GetName())
		}
		l = append(l[:index], l[index+1:]...)
	}
	return l, nil
}

func (r BaseAlertRule) GetNamespace() string {
	return r.Namespace
}

func (r BaseAlertRule) GetName() string {
	return r.Name
}

func (r BaseAlertRule) GetInhibitLabels() []string {
	return r.InhibitLabels
}

func (r BaseAlertRule) GetAlertLevels() []AlertLevel {
	return r.AlertLevels
}

func (r BaseAlertRule) GetReceivers() []AlertReceiver {
	return r.Receivers
}

type BaseAlertResource struct {
	AMConfig *v1alpha1.AlertmanagerConfig
	Silences []alertmanagertypes.Silence
}

func (base *BaseAlertResource) GetReceiverMap() (map[string][]AlertReceiver, error) {
	routes, err := base.AMConfig.Spec.Route.ChildRoutes()
	if err != nil {
		return nil, err
	}
	// 以 alert name 为 key
	ret := map[string][]AlertReceiver{}
	for _, route := range routes {
		for _, m := range route.Matchers {
			if m.Name == AlertNameLabel {
				ret[m.Value] = append(ret[m.Value], AlertReceiver{
					Name:     route.Receiver,
					Interval: route.RepeatInterval,
				})
			}
		}
	}
	return ret, nil
}

func (base *BaseAlertResource) GetSilenceMap() map[string]alertmanagertypes.Silence {
	ret := map[string]alertmanagertypes.Silence{}
	for _, s := range base.Silences {
		// 有fingerprint- 前缀的是告警黑名单
		if !strings.HasPrefix(s.Comment, "fingerprint-") {
			for _, matcher := range s.Matchers {
				if matcher.Name == AlertNameLabel {
					ret[matcher.Value] = s
				}
			}
		}
	}
	return ret
}

func (base *BaseAlertResource) GetInhibitRuleMap() map[string]v1alpha1.InhibitRule {
	ret := map[string]v1alpha1.InhibitRule{}
	for _, v := range base.AMConfig.Spec.InhibitRules {
		for _, m := range v.SourceMatch {
			if m.Name == AlertNameLabel {
				ret[m.Name] = v
			}
		}
	}
	return ret
}

func (base *BaseAlertResource) UpdateRoutes(alertrules AlertRuleList[AlertRule]) {
	base.AMConfig.Spec.Route.Routes = nil
	for _, alertrule := range alertrules {
		for _, receiver := range alertrule.GetReceivers() {
			rawRouteData, _ := json.Marshal(v1alpha1.Route{
				Receiver:       receiver.Name,
				RepeatInterval: receiver.Interval,
				Continue:       true,
				Matchers: []v1alpha1.Matcher{
					{
						Name:  AlertNamespaceLabel,
						Value: alertrule.GetNamespace(),
					},
					{
						Name:  AlertNameLabel,
						Value: alertrule.GetName(),
					},
				},
			})
			base.AMConfig.Spec.Route.Routes = append(base.AMConfig.Spec.Route.Routes, apiextensionsv1.JSON{Raw: rawRouteData})
		}
	}
	base.AMConfig.Spec.Route.Receiver = NullReceiverName
	base.AMConfig.Spec.Route.GroupBy = []string{AlertNamespaceLabel, AlertNameLabel}
	base.AMConfig.Spec.Route.GroupInterval = "30s" // ref. https://zhuanlan.zhihu.com/p/63270049. group_interval设短点好
	base.AMConfig.Spec.Route.GroupWait = "30s"     // 使用默认值
	base.AMConfig.Spec.Route.Matchers = nil
}

func (base *BaseAlertResource) UpdateInhibitRules(alertrules AlertRuleList[AlertRule]) error {
	base.AMConfig.Spec.InhibitRules = nil
	inhibitRuleMap := map[string]v1alpha1.InhibitRule{}
	for _, alertrule := range alertrules {
		// 更新AlertmanagerConfig inhibitRules
		// 先用map为同一label的去重
		if len(alertrule.GetAlertLevels()) > 1 {
			if len(alertrule.GetInhibitLabels()) == 0 {
				return fmt.Errorf("有多个告警级别时，告警抑制标签不能为空!")
			}
			inhibitRuleMap[slice.SliceUniqueKey(alertrule.GetInhibitLabels())] = v1alpha1.InhibitRule{
				SourceMatch: []v1alpha1.Matcher{
					{
						Name:  SeverityLabel,
						Value: SeverityCritical,
						Regex: false,
					},
				},
				TargetMatch: []v1alpha1.Matcher{
					{
						Name:  SeverityLabel,
						Value: SeverityError,
						Regex: false,
					},
				},
				Equal: append(alertrule.GetInhibitLabels(), AlertNamespaceLabel, AlertNameLabel),
			}
		}
	}

	for _, v := range inhibitRuleMap {
		base.AMConfig.Spec.InhibitRules = append(base.AMConfig.Spec.InhibitRules, v)
	}
	return nil
}

func (base *BaseAlertResource) AddNullReceivers() {
	// 检查并添加空接收器
	foundNull := false
	for _, v := range base.AMConfig.Spec.Receivers {
		if v.Name == NullReceiverName {
			foundNull = true
			continue
		}
	}
	if !foundNull {
		base.AMConfig.Spec.Receivers = append(base.AMConfig.Spec.Receivers, NullReceiver)
	}
}

func CheckAlertNameInAMConfig(name string, amconfig *v1alpha1.AlertmanagerConfig, msg string) error {
	routes, err := amconfig.Spec.Route.ChildRoutes()
	if err != nil {
		return err
	}
	for _, v := range routes {
		for _, m := range v.Matchers {
			if m.Name == AlertNameLabel && m.Value == name {
				return fmt.Errorf("已有同名的%s告警规则: %s", msg, name)
			}
		}
	}
	return nil
}

func getAlertScope(namespace string) string {
	if namespace == GlobalAlertNamespace {
		return ScopeSystemAdmin
	}
	return ScopeNormal
}
