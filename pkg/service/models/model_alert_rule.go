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
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"kubegems.io/kubegems/pkg/utils/gormdatatypes"
)

// AlertRule
type AlertRule struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	Cluster   string `gorm:"type:varchar(50)" json:"cluster"`
	Namespace string `gorm:"type:varchar(50)" json:"namespace"`
	Name      string `gorm:"type:varchar(50)" binding:"required" json:"name"`
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

func (r *AlertRule) FullName() string {
	return fmt.Sprintf("[cluster:%s, namespace: %s, name: %s]", r.Cluster, r.Namespace, r.Name)
}
