package online

import (
	"encoding/json"

	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/gemsplugin"
	"kubegems.io/pkg/utils/prometheus"
)

type OnlineConfig interface {
	ConfigName() string
	CheckOptions() error
}

var (
	_ OnlineConfig = &gemsplugin.InstallerOptions{}
	_ OnlineConfig = &prometheus.MonitorOptions{}
)

func LoadOptions(opts OnlineConfig, db *gorm.DB) error {
	config := models.OnlineConfig{}
	if err := db.First(&config, "name = ?", opts.ConfigName()).Error; err != nil {
		log.Error(err, "get config", "name", opts.ConfigName())
		return err
	}
	return json.Unmarshal(config.Content, opts)
}

func SaveOptions(opts OnlineConfig, db *gorm.DB) error {
	if err := opts.CheckOptions(); err != nil {
		log.Error(err, "check config", "name", opts.ConfigName())
		return err
	}

	bts, err := json.Marshal(opts)
	if err != nil {
		return err
	}
	config := models.OnlineConfig{
		Name:    opts.ConfigName(),
		Content: bts,
	}

	if err := db.Save(&config).Error; err != nil {
		log.Error(err, "save config", "name", opts.ConfigName())
	}
	return err
}
