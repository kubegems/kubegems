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

package options

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
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
