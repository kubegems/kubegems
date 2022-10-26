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
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	alertmanagertypes "github.com/prometheus/alertmanager/types"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/channels"
	"kubegems.io/kubegems/pkg/utils/set"
	"kubegems.io/kubegems/pkg/utils/slice"
)

type AlertLevel struct {
	CompareOp    string `json:"compareOp"`
	CompareValue string `json:"compareValue"` // 支持表达式, eg. 24 * 60
	Severity     string `json:"severity"`     // error, critical
}

type AlertReceiver struct {
	AlertChannel *models.AlertChannel `json:"alertChannel"`
	Interval     string               `json:"interval"` // 分组间隔
}

type BaseAlertRule struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`

	Expr    string `json:"expr"`    // promql/logql表达式，不能包含比较运算符(<, <=, >, >=, ==)
	For     string `json:"for"`     // 持续时间, eg. 10s, 1m, 1h
	Message string `json:"message"` // 告警消息，若为空后端自动填充

	InhibitLabels []string        `json:"inhibitLabels"` // 如果有多个告警级别，需要配置告警抑制的labels
	AlertLevels   []AlertLevel    `json:"alertLevels"`   // 告警级别
	Receivers     []AlertReceiver `json:"receivers"`     // 接收器

	IsOpen bool   `json:"isOpen"` // 是否启用
	State  string `json:"state"`  // 状态
}

// IsExtraAlert 用于记录额外信息
// 如在生成监控的amcfg时，避免忽视日志的route
func (r *BaseAlertRule) IsExtraAlert() bool {
	return len(r.AlertLevels) == 0
}

func CheckQueryExprNamespace(expr, namespace string) error {
	if namespace != "" && namespace != prometheus.GlobalAlertNamespace {
		if !(strings.Contains(expr, fmt.Sprintf(`namespace=~"%s"`, namespace)) ||
			strings.Contains(expr, fmt.Sprintf(`namespace="%s"`, namespace))) {
			return fmt.Errorf(`query expr %[1]s must contains namespace %[2]s, eg: {namespace="%[2]s"}`, expr, namespace)
		}
	}
	return nil
}

func (r *BaseAlertRule) CheckAndModify() error {
	_, _, _, hasOp := prometheus.SplitQueryExpr(r.Expr)
	if hasOp {
		return fmt.Errorf("查询表达式不能包含比较运算符(<|<=|==|!=|>|>=)")
	}
	if err := CheckQueryExprNamespace(r.Expr, r.Namespace); err != nil {
		return err
	}

	// check receivers
	if len(r.Receivers) == 0 {
		return fmt.Errorf("接收器不能为空")
	}
	receverSet := set.NewSet[string]()
	for _, rec := range r.Receivers {
		if receverSet.Has(rec.AlertChannel.ReceiverName()) {
			return fmt.Errorf("接收器: %s重复", rec.AlertChannel.ReceiverName())
		} else {
			receverSet.Append(rec.AlertChannel.ReceiverName())
		}
	}
	if !receverSet.Has(models.DefaultReceiver.Name) {
		r.Receivers = append(r.Receivers, AlertReceiver{
			AlertChannel: models.DefaultChannel,
			Interval:     r.Receivers[0].Interval,
		})
	}

	// check alert levels
	if len(r.AlertLevels) == 0 {
		return fmt.Errorf("告警级别不能为空")
	}
	severitySet := set.NewSet[string]()
	for _, v := range r.AlertLevels {
		if severitySet.Has(v.Severity) {
			return fmt.Errorf("有重复的告警级别")
		} else {
			severitySet.Append(v.Severity)
		}
	}

	if len(r.AlertLevels) > 1 && len(r.InhibitLabels) == 0 {
		return fmt.Errorf("有多个告警级别时，告警抑制标签不能为空!")
	}
	return nil
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
	channels.ChannelGetter
}

func (base *BaseAlertResource) GetAlertReceiverMap() (map[string][]AlertReceiver, error) {
	routes, err := base.AMConfig.Spec.Route.ChildRoutes()
	if err != nil {
		return nil, err
	}
	// 以 alert name 为 key
	ret := map[string][]AlertReceiver{}
	for _, route := range routes {
		for _, m := range route.Matchers {
			if m.Name == prometheus.AlertNameLabel {
				name, id := models.ChannelIDNameByReceiverName(route.Receiver)
				rec := AlertReceiver{
					Interval: route.RepeatInterval,
					AlertChannel: &models.AlertChannel{
						ID:   id,
						Name: name,
					},
				}
				ret[m.Value] = append(ret[m.Value], rec)
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
				if matcher.Name == prometheus.AlertNameLabel {
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
			if m.Name == prometheus.AlertNameLabel {
				ret[m.Value] = v
			}
		}
	}
	return ret
}

func (base *BaseAlertResource) Update(alertrules AlertRuleList[AlertRule]) error {
	// update AlertmanagerConfig routes
	base.UpdateRoutes(alertrules)
	// update AlertmanagerConfig inhibit rules
	base.UpdateInhibitRules(alertrules)
	// update AlertmanagerConfig receivers
	return base.UpdateReceivers(alertrules)
}

func (base *BaseAlertResource) UpdateReceivers(alertrules AlertRuleList[AlertRule]) error {
	base.AMConfig.Spec.Receivers = []v1alpha1.Receiver{
		prometheus.NullReceiver,
		models.DefaultReceiver,
	}
	recSet := set.NewSet[uint]().Append(models.DefaultChannel.ID)
	for _, alertrule := range alertrules {
		for _, rec := range alertrule.GetReceivers() {
			if !recSet.Has(rec.AlertChannel.ID) {
				ch, err := base.ChannelGetter(rec.AlertChannel.ID)
				if err != nil {
					return errors.Wrapf(err, "failed to get channel by receiver: %v", rec)
				}
				base.AMConfig.Spec.Receivers = append(base.AMConfig.Spec.Receivers, ch.ToReceiver(rec.AlertChannel.ReceiverName()))
				recSet.Append(rec.AlertChannel.ID)
			}
		}
	}
	return nil
}

func (base *BaseAlertResource) UpdateRoutes(alertrules AlertRuleList[AlertRule]) {
	base.AMConfig.Spec.Route.Routes = nil
	for _, alertrule := range alertrules {
		for _, receiver := range alertrule.GetReceivers() {
			rawRouteData, _ := json.Marshal(v1alpha1.Route{
				Receiver:       receiver.AlertChannel.ReceiverName(),
				RepeatInterval: receiver.Interval,
				Continue:       true,
				Matchers: []v1alpha1.Matcher{
					{
						Name:  prometheus.AlertNamespaceLabel,
						Value: alertrule.GetNamespace(),
					},
					{
						Name:  prometheus.AlertNameLabel,
						Value: alertrule.GetName(),
					},
				},
			})
			base.AMConfig.Spec.Route.Routes = append(base.AMConfig.Spec.Route.Routes, apiextensionsv1.JSON{Raw: rawRouteData})
		}
	}
	base.AMConfig.Spec.Route.Receiver = prometheus.NullReceiverName
	base.AMConfig.Spec.Route.GroupBy = []string{prometheus.AlertNamespaceLabel, prometheus.AlertNameLabel}
	base.AMConfig.Spec.Route.GroupInterval = "30s" // ref. https://zhuanlan.zhihu.com/p/63270049. group_interval设短点好
	base.AMConfig.Spec.Route.GroupWait = "30s"     // 使用默认值
	base.AMConfig.Spec.Route.Matchers = nil
}

func (base *BaseAlertResource) UpdateInhibitRules(alertrules AlertRuleList[AlertRule]) {
	base.AMConfig.Spec.InhibitRules = nil
	inhibitRuleMap := map[string]v1alpha1.InhibitRule{}
	for _, alertrule := range alertrules {
		// 更新AlertmanagerConfig inhibitRules
		// 先用map为同一label的去重
		if len(alertrule.GetInhibitLabels()) > 0 {
			inhibitRuleMap[slice.SliceUniqueKey(append(alertrule.GetInhibitLabels(), alertrule.GetNamespace(), alertrule.GetName()))] = v1alpha1.InhibitRule{
				SourceMatch: []v1alpha1.Matcher{
					{
						Name:  prometheus.AlertNamespaceLabel,
						Value: alertrule.GetNamespace(),
					},
					{
						Name:  prometheus.AlertNameLabel,
						Value: alertrule.GetName(),
					},
					{
						Name:  prometheus.SeverityLabel,
						Value: prometheus.SeverityCritical,
						Regex: false,
					},
				},
				TargetMatch: []v1alpha1.Matcher{
					{
						Name:  prometheus.AlertNamespaceLabel,
						Value: alertrule.GetNamespace(),
					},
					{
						Name:  prometheus.AlertNameLabel,
						Value: alertrule.GetName(),
					},
					{
						Name:  prometheus.SeverityLabel,
						Value: prometheus.SeverityError,
						Regex: false,
					},
				},
				Equal: append(alertrule.GetInhibitLabels(), prometheus.AlertNamespaceLabel, prometheus.AlertNameLabel),
			}
		}
	}

	for _, v := range inhibitRuleMap {
		base.AMConfig.Spec.InhibitRules = append(base.AMConfig.Spec.InhibitRules, v)
	}
}

func getAlertScope(namespace string) string {
	if namespace == prometheus.GlobalAlertNamespace {
		return prometheus.ScopeSystemAdmin
	}
	return prometheus.ScopeNormal
}
