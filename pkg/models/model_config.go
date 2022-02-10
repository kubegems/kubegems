package models

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/prometheus"
)

const (
	MetricConfig    = "metric"
	InstallerConfig = "installer"
)

type ConfigInDBInterface interface {
	Reload() error
}

func InitConfig(db *gorm.DB) {
	cfgs := []Config{}
	if err := db.Find(&cfgs).Error; err != nil {
		panic(err)
	}
	for i := range cfgs {
		if err := cfgs[i].DecodeContent(); err != nil {
			panic(err)
		}
		if err := cfgs[i].ContentObj.Reload(); err != nil {
			panic(err)
		}
		log.Info("config load succeed", "name", cfgs[i].Name)
	}
}

// MetricDashborad 监控面板
type Config struct {
	ID uint `gorm:"primarykey"`
	// 面板名
	Name    string `gorm:"type:varchar(50);uniqueIndex" binding:"required"`
	Content datatypes.JSON

	ContentObj ConfigInDBInterface `gorm:"-" json:"-"`
}

func (c *Config) DecodeContent() error {
	switch c.Name {
	case MetricConfig:
		tmp := prometheus.GemsMetricConfig{}
		if err := json.Unmarshal(c.Content, &tmp); err != nil {
			return err
		}
		c.ContentObj = tmp
	default:
		return fmt.Errorf("unknown config name: %s", c.Name)
	}
	return nil
}

func (c *Config) AfterSave(tx *gorm.DB) error {
	if err := c.DecodeContent(); err != nil {
		return err
	}

	return c.ContentObj.Reload()
}
