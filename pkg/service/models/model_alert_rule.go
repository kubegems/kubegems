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

package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"kubegems.io/kubegems/pkg/utils/gormdatatypes"
)

// AlertRule
type AlertRule struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	Cluster   string `gorm:"type:varchar(50)" json:"cluster"`
	Namespace string `gorm:"type:varchar(50)" json:"namespace"`
	Name      string `gorm:"type:varchar(50)" binding:"min=1,max=50" json:"name"`
	AlertType string `gorm:"type:varchar(50);default:monitor" json:"alertType"` // logging or monitor

	Expr    string `json:"expr"`                                           // promql/logql表达式，不能包含比较运算符(<, <=, >, >=, ==)
	For     string `gorm:"type:varchar(50)" binding:"required" json:"for"` // 持续时间, eg. 10s, 1m, 1h
	Message string `json:"message"`                                        // 告警消息

	InhibitLabels gormdatatypes.JSONSlice `json:"inhibitLabels"`                                                 // 如果有多个告警级别，需要配置告警抑制的labels
	AlertLevels   AlertLevels             `json:"alertLevels"`                                                   // 告警级别
	Receivers     []*AlertReceiver        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"receivers"` // 接收器, 删除alertrule时级联删除

	PromqlGenerator *PromqlGenerator `json:"promqlGenerator"`
	LogqlGenerator  *LogqlGenerator  `json:"logqlGenerator"`

	IsOpen         bool            `gorm:"default:true" json:"isOpen"` // 是否启用
	State          string          `json:"state"`                      // 状态
	RealTimeAlerts []*promv1.Alert `gorm:"-" json:"realTimeAlerts"`    // 实时告警

	// eg: status: ok/error
	// reason: alertmanagerconfig lost/receiver not matched/...
	K8sResourceStatus gormdatatypes.JSONMap `gorm:"type:varchar(50)" json:"k8sResourceStatus"` // 对应的k8s资源状态

	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

type AlertReceiver struct {
	ID uint `gorm:"primarykey" json:"id"`

	AlertRuleID uint       `json:"alertRuleID"`
	AlertRule   *AlertRule `json:"-"` // 不展示给前端

	AlertChannelID uint          `json:"alertChannelID"`
	AlertChannel   *AlertChannel `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"alertChannel"` // 删除channel时RESTRICT拒绝

	Interval string `json:"interval"`
}

func AlertRuleKey(cluster, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", cluster, namespace, name)
}

func (r *AlertRule) FullName() string {
	return AlertRuleKey(r.Cluster, r.Namespace, r.Name)
}

// [metadata.name: Invalid value: "艹": a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*'), metadata.labels: Invalid value: "艹": a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')]
func IsValidAlertRuleName(name string) error {
	errs := validation.IsDNS1035Label(name)
	errs = append(errs, validation.IsValidLabelValue(name)...)
	if len(errs) > 0 {
		return errors.Errorf("alert rule name not valid: %s", strings.Join(errs, ", "))
	}
	return nil
}
