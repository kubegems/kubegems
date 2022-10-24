package prometheus

import (
	"database/sql/driver"
	"encoding/json"
)

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
	*PromqlGenerator `json:"promqlGenerator"`
	Expr             string `json:"expr"`
	Unit             string `json:"unit"`
}
