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
	"fmt"
	"regexp"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"kubegems.io/kubegems/pkg/apis/gems"
)

const (
	AlertNamespaceLabel = "gems_namespace"
	AlertNameLabel      = "gems_alertname"
	// 用于从告警中获取告警资源
	AlertFromLabel = "gems_alert_from" // 告警来自哪里，logging/monitor
	AlertPromqlTpl = "gems_alert_tpl"  // eg. platform.cluster.cpuUsageTotal

	AlertTypeMonitor = "monitor"
	AlertTypeLogging = "logging"

	SeverityLabel    = "severity"
	SeverityError    = "error"    // 错误
	SeverityCritical = "critical" // 严重

	ExprJsonAnnotationKey = "gems_expr_json"
	MessageAnnotationsKey = "message"
	ValueAnnotationKey    = "value"
	ValueAnnotationExpr   = `{{ $value | printf "%.1f" }}`

	AlertRuleKeyFormat = "gems-%s-%s"
	AlertClusterKey    = "cluster"

	// 告警消息发送范围
	AlertScopeLabel  = "gems_alert_scope"
	ScopeSystemAdmin = "system-admin" // 系统管理员
	ScopeSystemUser  = "system-user"  // 所有用户
	ScopeNormal      = "normal"       // 普通租户用户

	SilenceCommentForBlackListPrefix = "fingerprint-"
	SilenceCommentForAlertrulePrefix = "silence for"
	// 全局告警命名空间，非此命名空间强制加上namespace筛选
	GlobalAlertNamespace = gems.NamespaceMonitor
	// namespace
	PromqlNamespaceKey = "namespace"

	// prometheusrule and alertmanagerconfigname
	DefaultAlertCRDName = "kubegems-default-monitor-alert-rule"
)

type RealTimeAlertRule struct {
	Name string `json:"name"`
	// Query          string         `json:"query"`
	// Duration       float64        `json:"duration"`
	// Labels         model.LabelSet `json:"labels"`
	// Annotations    model.LabelSet `json:"annotations"`
	Alerts []*v1.Alert `json:"alerts"`
	// Health         v1.RuleHealth  `json:"health"`
	// LastError      string         `json:"lastError,omitempty"`
	// EvaluationTime float64        `json:"evaluationTime"`
	// LastEvaluation time.Time      `json:"lastEvaluation"`
	State string `json:"state"`
	// Type           v1.RuleType    `json:"type"`
}

func (r *RealTimeAlertRule) Len() int      { return len(r.Alerts) }
func (r *RealTimeAlertRule) Swap(i, j int) { r.Alerts[i], r.Alerts[j] = r.Alerts[j], r.Alerts[i] }
func (r *RealTimeAlertRule) Less(i, j int) bool {
	return r.Alerts[i].ActiveAt.After(r.Alerts[j].ActiveAt)
} // 倒排

func RealTimeAlertKey(namespace, name string) string {
	return fmt.Sprintf(AlertRuleKeyFormat, namespace, name)
}

var exprReg = regexp.MustCompile("(.*)(<|<=|==|!=|>|>=)(.*)")

func SplitQueryExpr(ql string) (query, op, value string, hasOp bool) {
	substrs := exprReg.FindStringSubmatch(ql)
	if len(substrs) == 4 {
		query = substrs[1]
		op = substrs[2]
		value = substrs[3]
		hasOp = true
	} else {
		query = ql
	}
	return
}

type WebhookAlert struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	Alerts            []Alert           `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int64             `json:"truncatedAlerts"`
}

func (w *WebhookAlert) FingerprintMap() map[string][]Alert {
	ret := map[string][]Alert{}
	for _, v := range w.Alerts {
		alerts, ok := ret[v.Fingerprint]
		if ok {
			alerts = append(alerts, v)
		} else {
			alerts = []Alert{v}
		}
		ret[v.Fingerprint] = alerts
	}

	return ret
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     *time.Time        `json:"startsAt"`
	EndsAt       *time.Time        `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}
