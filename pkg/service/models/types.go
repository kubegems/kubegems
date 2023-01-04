package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"kubegems.io/kubegems/pkg/utils/prometheus/promql"
	"kubegems.io/kubegems/pkg/utils/prometheus/templates"
)

type AlertLevels []AlertLevel
type AlertLevel struct {
	CompareOp    string `json:"compareOp"`
	CompareValue string `json:"compareValue"` // 支持表达式, eg. 24 * 60
	Severity     string `json:"severity"`     // error, critical
}

func (m AlertLevels) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	ba, err := json.Marshal(m)
	return string(ba), err
}

func (m *AlertLevels) Scan(val interface{}) error {
	if val == nil {
		*m = make(AlertLevels, 0)
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", val))
	}
	t := AlertLevels{}
	err := json.Unmarshal(ba, &t)
	*m = t
	return err
}

func (m AlertLevels) GormDataType() string {
	return "json"
}

type PromqlGenerator struct {
	Scope         string                `json:"scope"`         // scope
	Resource      string                `json:"resource"`      // 告警资源, eg. node、pod
	Rule          string                `json:"rule"`          // 告警规则名, eg. cpuUsage、memoryUsagePercent
	Unit          string                `json:"unit"`          // 单位
	LabelMatchers []promql.LabelMatcher `json:"labelMatchers"` // 标签筛选器

	Tpl *templates.PromqlTpl `json:"-"`
}

func (m PromqlGenerator) Value() (driver.Value, error) {
	promql.RemoveDuplicated(m.LabelMatchers)
	ba, err := json.Marshal(m)
	return string(ba), err
}

func (m *PromqlGenerator) Scan(val interface{}) error {
	if val == nil {
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", val))
	}
	t := PromqlGenerator{}
	err := json.Unmarshal(ba, &t)
	*m = t
	return err
}

func (m PromqlGenerator) GormDataType() string {
	return "json"
}

type LogqlGenerator struct {
	Duration      string                `json:"duration"`      // 时间范围
	Match         string                `json:"match"`         // 正则匹配的字符串
	LabelMatchers []promql.LabelMatcher `json:"labelMatchers"` // 标签筛选器
}

func (m LogqlGenerator) Value() (driver.Value, error) {
	promql.RemoveDuplicated(m.LabelMatchers)
	ba, err := json.Marshal(m)
	return string(ba), err
}

func (m *LogqlGenerator) Scan(val interface{}) error {
	if val == nil {
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", val))
	}
	t := LogqlGenerator{}
	err := json.Unmarshal(ba, &t)
	*m = t
	return err
}

func (m LogqlGenerator) GormDataType() string {
	return "json"
}
