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
	"kubegems.io/kubegems/pkg/model/store"
	microservice "kubegems.io/kubegems/pkg/service/handlers/microservice/options"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/helm"
	"kubegems.io/kubegems/pkg/utils/jwt"
	"kubegems.io/kubegems/pkg/utils/msgbus"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/utils/system"
)

type Options struct {
	System       *system.Options                   `json:"system,omitempty"`
	Appstore     *helm.Options                     `json:"appstore,omitempty"`
	Argo         *argo.Options                     `json:"argo,omitempty"`
	DebugMode    bool                              `json:"debugMode,omitempty"`
	Exporter     *prometheus.ExporterOptions       `json:"exporter,omitempty"`
	Git          *git.Options                      `json:"git,omitempty"`
	JWT          *jwt.Options                      `json:"jwt,omitempty"`
	LogLevel     string                            `json:"logLevel,omitempty"`
	Msgbus       *msgbus.Options                   `json:"msgbus,omitempty"`
	Mysql        *database.Options                 `json:"mysql,omitempty"`
	Redis        *redis.Options                    `json:"redis,omitempty"`
	Microservice *microservice.MicroserviceOptions `json:"microservice,omitempty"`
	Mongo        *store.MongoDBOptions             `json:"mongo,omitempty"`
}

func DefaultOptions() *Options {
	defaultoptions := &Options{
		Appstore:     helm.NewDefaultOptions(),
		Argo:         argo.NewDefaultArgoOptions(),
		DebugMode:    false,
		Exporter:     prometheus.DefaultExporterOptions(),
		Git:          git.NewDefaultOptions(),
		JWT:          jwt.DefaultOptions(),
		LogLevel:     "debug",
		Msgbus:       msgbus.DefaultMsgbusOptions(),
		Mysql:        database.NewDefaultOptions(),
		Redis:        redis.NewDefaultOptions(),
		System:       system.NewDefaultOptions(),
		Microservice: microservice.NewDefaultOptions(),
		Mongo:        store.DefaultStoreOptions().MongoDB,
	}
	defaultoptions.System.Listen = ":8020"
	return defaultoptions
}
