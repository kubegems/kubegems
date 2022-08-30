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
	"database/sql/driver"
	"encoding/json"
	"time"

	"kubegems.io/kubegems/pkg/utils/gormdatatypes"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

// MonitorDashboard 监控面板
type MonitorDashboard struct {
	ID uint `gorm:"primarykey" json:"id"`
	// 面板名
	Name      string                `gorm:"type:varchar(50);uniqueIndex" binding:"required" json:"name"`
	Step      string                `gorm:"type:varchar(50)" json:"step"`    // 样本间隔，单位秒
	Refresh   string                `gorm:"type:varchar(50)" json:"refresh"` // 刷新间隔，eg. 30s, 1m
	Start     string                `gorm:"type:varchar(50)" json:"start"`   // 开始时间，eg. 2022-04-24 06:00:45.241, now, now-30m
	End       string                `gorm:"type:varchar(50)" json:"end"`     // 结束时间
	CreatedAt *time.Time            `json:"createdAt"`
	Creator   string                `gorm:"type:varchar(50)" json:"creator"` // 创建者
	Graphs    MonitorGraphs         `json:"graphs"`                          // 图表
	Variables gormdatatypes.JSONMap `json:"variables"`                       // 变量

	Template string `gorm:"type:varchar(50)" json:"template"` // 模板名

	EnvironmentID *uint        `json:"environmentID"`
	Environment   *Environment `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;" json:"environment"`
}

type MonitorDashboardTpl struct {
	Name        string                `gorm:"type:varchar(50);primaryKey" json:"name"`
	Description string                `json:"description"`
	Step        string                `gorm:"type:varchar(50)" json:"step"`    // 样本间隔，单位秒
	Refresh     string                `gorm:"type:varchar(50)" json:"refresh"` // 刷新间隔，eg. 30s, 1m
	Start       string                `gorm:"type:varchar(50)" json:"start"`   // 开始时间，eg. 2022-04-24 06:00:45.241, now, now-30m
	End         string                `gorm:"type:varchar(50)" json:"end"`     // 结束时间
	Graphs      MonitorGraphs         `json:"graphs"`                          // 图表
	Variables   gormdatatypes.JSONMap `json:"variables"`                       // 变量

	CreatedAt *time.Time `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// 实现几个自定义数据类型的接口 https://gorm.io/zh_CN/docs/data_types.html
func (g *MonitorGraphs) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, g)
	}
	return nil
}

// 注意这里不是指针，下同
func (g MonitorGraphs) Value() (driver.Value, error) {
	return json.Marshal(g)
}

func (g MonitorGraphs) GormDataType() string {
	return "json"
}

type MonitorGraphs []MetricGraph

type MetricGraph struct {
	// graph名
	Name string `json:"name"`
	// 查询目标
	*prometheus.PromqlGenerator `json:"promqlGenerator"`
	Expr                        string `json:"expr"`
	Unit                        string `json:"unit"`
}
