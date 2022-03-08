package options

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/models"
)

type DynamicOptions interface {
	Name() string
	Validate() error
}

type DynamicConfigurationProviderIface interface {
	Get(ctx context.Context, opts DynamicOptions) error
	Set(ctx context.Context, opts DynamicOptions) error
}

type DatabaseDynamicConfigurationProvider struct {
	db *gorm.DB
}

func NewDatabaseDynamicConfigurationProvider(db *gorm.DB) DynamicConfigurationProviderIface {
	return &DatabaseDynamicConfigurationProvider{
		db: db,
	}
}

func (p *DatabaseDynamicConfigurationProvider) Get(ctx context.Context, opts DynamicOptions) error {
	config := models.OnlineConfig{}
	if err := p.db.WithContext(ctx).First(&config, "name = ?", opts.Name()).Error; err != nil {
		log.Error(err, "failed to load configuration from db dynamic configuration provider", "config name", opts.Name())
		return err
	}
	return json.Unmarshal(config.Content, opts)
}

func (p *DatabaseDynamicConfigurationProvider) Set(ctx context.Context, opts DynamicOptions) error {
	if err := opts.Validate(); err != nil {
		log.Error(err, "validate configuration failed", "config name", opts.Name())
		return err
	}

	bts, err := json.Marshal(opts)
	if err != nil {
		return err
	}
	config := models.OnlineConfig{
		Name:    opts.Name(),
		Content: bts,
	}

	if err := p.db.WithContext(ctx).Save(&config).Error; err != nil {
		log.Error(err, "failed to set dynamic configuration", "config name", opts.Name())
	}
	return nil
}
