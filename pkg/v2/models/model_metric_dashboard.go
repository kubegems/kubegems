package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"kubegems.io/kubegems/pkg/utils/prometheus"
)

// MetricDashborad 监控面板
type MetricDashborad struct {
	ID        uint   `gorm:"primarykey"`
	Name      string `gorm:"type:varchar(50);uniqueIndex" binding:"required"`
	Topk      int
	Step      string `gorm:"type:varchar(50);"`
	CreatedAt *time.Time
	Creator   string `gorm:"type:varchar(50);"` // 创建者
	Graphs    MetricGraphs
}

// 实现几个自定义数据类型的接口 https://gorm.io/zh_CN/docs/data_types.html
func (g *MetricGraphs) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, g)
	}
	return nil
}

// 注意这里不是指针，下同
func (g MetricGraphs) Value() (driver.Value, error) {
	return json.Marshal(g)
}

func (g MetricGraphs) GormDataType() string {
	return "json"
}

type MetricGraphs []MetricGraph

type MetricGraph struct {
	// graph名
	Name string `json:"name"`

	// 查询范围
	Cluster       string `json:"cluster,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	EnvironmentID string `json:"environment_id,omitempty"`

	// 查询目标
	prometheus.BaseQueryParams
}
