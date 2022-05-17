package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"kubegems.io/pkg/utils/prometheus"
)

// MonitorDashboard 监控面板
type MonitorDashboard struct {
	ID uint `gorm:"primarykey" json:"id"`
	// 面板名
	Name      string        `gorm:"type:varchar(50);uniqueIndex" binding:"required" json:"name"`
	Step      string        `gorm:"type:varchar(50);" json:"step"` // 样本间隔，单位秒
	Refresh   string        `json:"refresh"`                       // 刷新间隔，eg. 30s, 1m
	Start     string        `json:"start"`                         // 开始时间，eg. 2022-04-24 06:00:45.241, now, now-30m
	End       string        `json:"end"`                           // 结束时间
	CreatedAt *time.Time    `json:"createdAt"`
	Creator   string        `gorm:"type:varchar(50);" json:"creator"` // 创建者
	Graphs    MonitorGraphs `json:"graphs"`                           // 图表

	EnvironmentID *uint
	Environment   *Environment `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
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
}
