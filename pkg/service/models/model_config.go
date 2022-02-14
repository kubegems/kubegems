package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/prometheus"
)

const (
	monitorConfigName = "monitor"
	smtpConfigName    = "smtp"
)

var (
	defaultMetricConfig = Config{
		ID:          1,
		Name:        monitorConfigName,
		Description: "监控告警",
		Items: map[string]ConfigItem{
			"metrics.yaml": {
				Title:       "监控告警规则",
				Required:    true,
				Description: "监控告警规则配置yaml",
				InputType:   Yaml,
				Content:     prometheus.DefaultMetricConfigContent(),
			},
		},
	}
	defaultSMTPConfig = Config{
		ID:          2,
		Name:        smtpConfigName,
		Description: "SMTP服务",
		Items: map[string]ConfigItem{
			"smtpServer": {
				Title:       "SMTP服务器",
				Required:    true,
				Description: "SMTP服务器地址(包含端口号)",
				Content:     "localhost:25",
				InputType:   Input,
			},
			"requireTLS": {
				Title:       "是否要求TLS",
				Required:    true,
				Description: "是否要求TLS",
				Content:     false,
				InputType:   CheckBox,
			},
			"from": {
				Title:       "发件人",
				Required:    true,
				Description: "发件人邮箱",
				Content:     "bob@gmail.com",
				InputType:   Input,
			},
			"authPassword": {
				Title:       "密码",
				Required:    true,
				Description: "发件人邮箱密码",
				Content:     "password",
				InputType:   Hidden,
			},
		},
	}
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
		if cfgs[i].Name == monitorConfigName {
			if item, ok := cfgs[i].Items["metrics.yaml"]; ok {
				// content取出来是map[string]interface{}，要转为struct才能reload
				bts, err := json.Marshal(item.Content)
				if err != nil {
					panic(err)
				}
				metricCfg := prometheus.GemsMetricConfig{}
				if err := json.Unmarshal(bts, &metricCfg); err != nil {
					panic(err)
				}
				if err := metricCfg.Reload(); err != nil {
					panic(err)
				}
				log.Info("config load succeed", "name", cfgs[i].Name)
			}
		}
	}
}

// Config 系统配置
type Config struct {
	ID          uint        `gorm:"primarykey"`
	Name        string      `gorm:"type:varchar(50);uniqueIndex" binding:"required"` // 配置名
	Description string      // 描述
	Items       ConfigItems // 配置项
}

type ConfigItems map[string]ConfigItem

type InputType string

const (
	Input    InputType = "text"     // 普通输入
	Hidden   InputType = "hidden"   // 隐藏显示输入，点击展示/隐藏
	CheckBox InputType = "checkbox" // 勾选框
	Yaml     InputType = "yaml"     // yaml文件
)

type ConfigItem struct {
	Title       string              `json:"title"`       // 配置项名
	Required    bool                `json:"required"`    // 是否必须
	Description string              `json:"description"` // 描述
	Content     interface{}         `json:"value"`       // 配置内容
	InputType   `json:"input_type"` // 输入方式
}

// 实现几个自定义数据类型的接口 https://gorm.io/zh_CN/docs/data_types.html
func (c *ConfigItems) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, c)
	}
	return nil
}

// 注意这里不是指针，下同
func (c ConfigItems) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c ConfigItems) GormDataType() string {
	return "json"
}

// 由于metric做了缓存，需要reload
func (c *Config) AfterSave(tx *gorm.DB) error {
	if c.Name == monitorConfigName {
		if item, ok := c.Items["metrics.yaml"]; ok {
			switch v := item.Content.(type) {
			case prometheus.GemsMetricConfig:
				return v.Reload()
			default:
				return fmt.Errorf("unknown config name: %s", c.Name)
			}
		}
	}
	return nil
}
