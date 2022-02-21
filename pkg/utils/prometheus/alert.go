package prometheus

import (
	"encoding/json"
	"fmt"
	"strings"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	alertmanagertypes "github.com/prometheus/alertmanager/types"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils"
)

const (
	AlertNamespaceLabel = "gems_namespace"
	AlertNameLabel      = "gems_alertname"
	// 用于从告警中获取告警资源
	AlertResourceLabel = "gems_alert_resource"
	AlertRuleLabel     = "gems_alert_rule"

	severityLabel    = "severity"
	severityError    = "error"    // 错误
	severityCritical = "critical" // 严重

	exprJsonAnnotationKey = "gems_expr_json"
	messageAnnotationsKey = "message"
	valueAnnotationKey    = "value"

	alertRuleKeyFormat = "gems-%s-%s"
	AlertClusterKey    = "cluster"

	// 告警消息发送范围
	AlertScopeLabel  = "gems_alert_scope"
	ScopeSystemAdmin = "system-admin" // 系统管理员
	ScopeSystemUser  = "system-user"  // 所有用户
	ScopeNormal      = "normal"       // 普通租户用户
)

// 里面资源的namespace必须相同
type RawAlertResource struct {
	*v1alpha1.AlertmanagerConfig
	*monitoringv1.PrometheusRule
	Silences []alertmanagertypes.Silence
}

type AlertLevel struct {
	CompareOp    string `json:"compareOp"`
	CompareValue string `json:"compareValue"` // 支持表达式, eg. 24 * 60
	Severity     string `json:"severity"`
}

type Receiver struct {
	Name     string `json:"name"`
	Interval string `json:"interval"` // 分组间隔
}

type AlertRule struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`

	BaseQueryParams `json:",inline"`
	For             string `json:"for"`     // 持续时间, eg. 10s, 1m, 1h
	Message         string `json:"message"` // 告警消息，若为空后端自动填充

	AlertLevels []AlertLevel `json:"alertLevels"` // 告警级别
	Receivers   []Receiver   `json:"receivers"`   // 接收器

	IsOpen bool   `json:"isOpen"` // 是否启用
	State  string `json:"state"`  // 状态

	RealTimeAlerts []*promv1.Alert `json:"realTimeAlerts,omitempty"` // 实时告警

	Promql string `json:"promql"` // 仅用于告知前端展示
	// 相关配置
	RuleContext `json:"-"`
	Origin      *monitoringv1.RuleGroup `json:"origin,omitempty"` // 原始的prometheusrule
}

// TODO: unit test
func (r *AlertRule) CheckAndModify() error {
	cfg := GetGemsMetricConfig(true)

	// check resource
	res, ok := cfg.Resources[r.Resource]
	if !ok {
		return fmt.Errorf("invalid resource: %s", r.Resource)
	}

	// check rule
	ruleCtx, err := r.FindRuleContext(cfg)
	if err != nil {
		return err
	}
	r.RuleContext = ruleCtx

	// check AlertLevels
	if len(r.AlertLevels) == 0 {
		return fmt.Errorf("alert level can't be null")
	}
	for _, v := range r.AlertLevels {
		if !utils.ContainStr(cfg.Operators, v.CompareOp) {
			return fmt.Errorf("invalid operator: %s", v.CompareOp)
		}
		if _, ok := cfg.Severity[v.Severity]; !ok {
			return fmt.Errorf("invalid severity: %s", v.Severity)
		}
	}

	// format message
	if r.Message == "" {
		r.Message = fmt.Sprintf("%s: [集群:{{ $externalLabels.%s }}] ", r.Name, AlertClusterKey)
		for _, label := range r.RuleDetail.Labels {
			r.Message += fmt.Sprintf("[%s:{{ $labels.%s }}] ", label, label)
		}

		r.Message += fmt.Sprintf("%s%s 触发告警, 当前值: %s%s", res.ShowName, r.RuleDetail.ShowName, `{{ $value | printf "%.1f" }}`, cfg.Units[r.Unit])
	}

	return r.checkAndModifyReceivers()
}

func (r *AlertRule) checkAndModifyReceivers() error {
	if len(r.Receivers) == 0 {
		return fmt.Errorf("接收器不能为空")
	}

	set := map[string]struct{}{}
	found := false
	for _, rec := range r.Receivers {
		if rec.Name == DefaultReceiverName {
			found = true
		}
		if _, ok := set[rec.Name]; ok {
			return fmt.Errorf("接收器: %s重复", rec.Name)
		} else {
			set[rec.Name] = struct{}{}
		}
	}

	if !found {
		r.Receivers = append(r.Receivers, Receiver{
			Name:     DefaultReceiverName,
			Interval: r.Receivers[0].Interval,
		})
	}
	return nil
}

// 默认认为namespace全部一致
func (raw *RawAlertResource) ToAlerts(containOrigin bool) ([]AlertRule, error) {
	receiverMap := map[string][]Receiver{}
	routes, err := raw.AlertmanagerConfig.Spec.Route.ChildRoutes()
	if err != nil {
		return nil, err
	}
	// 以 alert name 为 key
	for _, route := range routes {
		for _, m := range route.Matchers {
			if m.Name == AlertNameLabel {
				receiverMap[m.Value] = append(receiverMap[m.Value], Receiver{
					Name:     route.Receiver,
					Interval: route.RepeatInterval,
				})
			}
		}
	}

	silenceMap := map[string]alertmanagertypes.Silence{}
	for _, s := range raw.Silences {
		// 有fingerprint- 前缀的是告警黑名单
		if !strings.HasPrefix(s.Comment, "fingerprint-") {
			for _, matcher := range s.Matchers {
				if matcher.Name == AlertNameLabel {
					silenceMap[matcher.Value] = s
				}
			}
		}
	}

	ret := []AlertRule{}
	for i, group := range raw.PrometheusRule.Spec.Groups {
		alertname := group.Name

		// expr规则
		alertrule, err := convertAlertRule(raw.PrometheusRule.Namespace, group)
		if err != nil {
			log.Error(err, "convert prometheus rule")
			return nil, err
		}
		// 接收器
		alertrule.Receivers = receiverMap[alertname]

		// 是否启用
		isOpen := true
		if _, ok := silenceMap[alertname]; ok {
			isOpen = false
		}
		alertrule.IsOpen = isOpen
		if containOrigin {
			alertrule.Origin = &raw.PrometheusRule.Spec.Groups[i]
		}
		ret = append(ret, alertrule)
	}

	return ret, nil
}

// 所有alertrule都是一个namespace
func (raw *RawAlertResource) UpdateAlertRules(alertRules []AlertRule) error {
	groups := make([]monitoringv1.RuleGroup, len(alertRules))
	routes := []apiextensionsv1.JSON{}
	inhibitRuleMap := map[string]v1alpha1.InhibitRule{}
	for i, alertRule := range alertRules {
		// 更新 PrometheusRule
		groups[i].Name = alertRule.Name
		for _, level := range alertRule.AlertLevels {
			expr := CompareQueryParams{
				BaseQueryParams: alertRule.BaseQueryParams,
				CompareOp:       level.CompareOp,
				CompareValue:    level.CompareValue,
			}
			bts, _ := json.Marshal(expr)

			promql, err := expr.ConstructPromql(alertRule.Namespace)
			if err != nil {
				return err
			}
			groups[i].Rules = append(groups[i].Rules, monitoringv1.Rule{
				Alert: alertRule.Name,
				Expr:  intstr.FromString(promql),
				For:   alertRule.For,
				Labels: map[string]string{
					AlertNamespaceLabel: alertRule.Namespace,
					AlertNameLabel:      alertRule.Name,
					AlertResourceLabel:  alertRule.Resource,
					AlertRuleLabel:      alertRule.Rule,
					AlertScopeLabel:     GetAlertScope(alertRule.Namespace),
					severityLabel:       level.Severity,
				},
				Annotations: map[string]string{
					messageAnnotationsKey: alertRule.Message,
					valueAnnotationKey:    `{{ $value | printf "%.1f" }}`,
					exprJsonAnnotationKey: string(bts),
				},
			})
		}

		// 更新AlertmanagerConfig routes
		for _, receiver := range alertRule.Receivers {
			rawRouteData, _ := json.Marshal(v1alpha1.Route{
				Receiver:       receiver.Name,
				RepeatInterval: receiver.Interval,
				Continue:       true,
				Matchers: []v1alpha1.Matcher{
					{
						Name:  AlertNamespaceLabel,
						Value: alertRule.Namespace,
					},
					{
						Name:  AlertNameLabel,
						Value: alertRule.Name,
					},
				},
			})
			routes = append(routes, apiextensionsv1.JSON{Raw: rawRouteData})
		}

		// 更新AlertmanagerConfig inhibitRules
		// 先用map为同一label的去重
		if len(alertRule.AlertLevels) > 1 {
			inhibitRuleMap[utils.SliceUniqueKey(alertRule.RuleDetail.Labels)] = v1alpha1.InhibitRule{
				SourceMatch: []v1alpha1.Matcher{
					{
						Name:  severityLabel,
						Value: severityCritical,
						Regex: false,
					},
				},
				TargetMatch: []v1alpha1.Matcher{
					{
						Name:  severityLabel,
						Value: severityError,
						Regex: false,
					},
				},
				Equal: append(alertRule.RuleDetail.Labels, AlertNamespaceLabel, AlertNameLabel),
			}
		}
	}
	// 添加inhibitrule
	raw.AlertmanagerConfig.Spec.InhibitRules = nil
	for _, v := range inhibitRuleMap {
		raw.AlertmanagerConfig.Spec.InhibitRules = append(raw.AlertmanagerConfig.Spec.InhibitRules, v)
	}

	// 检查并添加空接收器
	foundNull := false
	for _, v := range raw.AlertmanagerConfig.Spec.Receivers {
		if v.Name == NullReceiverName {
			foundNull = true
			continue
		}
	}
	if !foundNull {
		raw.AlertmanagerConfig.Spec.Receivers = append(raw.AlertmanagerConfig.Spec.Receivers, NullReceiver)
	}

	raw.PrometheusRule.Spec.Groups = groups
	raw.AlertmanagerConfig.Spec.Route.Routes = routes
	raw.AlertmanagerConfig.Spec.Route.Receiver = NullReceiverName
	raw.AlertmanagerConfig.Spec.Route.GroupBy = []string{AlertNamespaceLabel, AlertNameLabel}
	raw.AlertmanagerConfig.Spec.Route.GroupInterval = "30s" // ref. https://zhuanlan.zhihu.com/p/63270049. group_interval设短点好
	raw.AlertmanagerConfig.Spec.Route.GroupWait = "30s"     // 使用默认值
	raw.AlertmanagerConfig.Spec.Route.Matchers = nil

	return nil
}

func (raw *RawAlertResource) ModifyAlertRule(newAlertRule AlertRule, act Action) error {
	alerts, err := raw.ToAlerts(false)
	if err != nil {
		return err
	}

	index := -1
	for i := range alerts {
		if alerts[i].Name == newAlertRule.Name {
			index = i
			break
		}
	}

	switch act {
	case Add:
		if index != -1 { // found
			return fmt.Errorf("告警规则 %s 已存在！", newAlertRule.Name)
		}
		alerts = append(alerts, newAlertRule)
	case Update:
		if index == -1 { // not found
			return fmt.Errorf("告警规则 %s 不存在！", newAlertRule.Name)
		}
		alerts[index] = newAlertRule
	case Delete:
		if index == -1 { // not found
			return fmt.Errorf("告警规则 %s 不存在！", newAlertRule.Name)
		}
		alerts = append(alerts[:index], alerts[index+1:]...)
	}

	return raw.UpdateAlertRules(alerts)
}

// TODO: unit test
func convertAlertRule(namespace string, group monitoringv1.RuleGroup) (AlertRule, error) {
	ret := AlertRule{}
	tmpexpr := CompareQueryParams{}
	for _, rule := range group.Rules {
		if rule.Labels[AlertNamespaceLabel] != namespace ||
			rule.Labels[AlertNameLabel] == "" {
			return ret, fmt.Errorf("rule label not valid: %v", rule.Labels)
		}

		exprJson := rule.Annotations[exprJsonAnnotationKey]
		if exprJson == "" {
			return ret, fmt.Errorf("rule annotation not contains expr json key")
		}

		expr := CompareQueryParams{}
		if err := json.Unmarshal([]byte(exprJson), &expr); err != nil {
			return ret, err
		}

		ret.Namespace = namespace
		ret.Name = rule.Labels[AlertNameLabel]

		ret.BaseQueryParams = expr.BaseQueryParams
		ret.For = rule.For
		ret.Message = rule.Annotations[messageAnnotationsKey]

		ret.AlertLevels = append(ret.AlertLevels, AlertLevel{
			CompareOp:    expr.CompareOp,
			CompareValue: expr.CompareValue,
			Severity:     rule.Labels[severityLabel],
		})

		tmpexpr.BaseQueryParams = expr.BaseQueryParams
	}

	// 填入ruleCtx
	ruleCtx, err := ret.FindRuleContext(GetGemsMetricConfig(true))
	if err != nil {
		return ret, err
	}
	ret.RuleContext = ruleCtx

	// 填入promql
	promql, err := tmpexpr.ConstructPromql(namespace)
	if err != nil {
		return ret, nil
	}
	ret.Promql = promql

	return ret, err
}

func RealTimeAlertKey(namespace, name string) string {
	return fmt.Sprintf(alertRuleKeyFormat, namespace, name)
}

func GetAlertScope(namespace string) string {
	if namespace == GlobalAlertNamespace {
		return ScopeSystemAdmin
	}
	return ScopeNormal
}
