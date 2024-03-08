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

package worker

import (
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/helm"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/worker/dump"
)

type Options struct {
	Listen   string                      `json:"listen,omitempty"`
	AppStore *helm.Options               `json:"appStore,omitempty"`
	Argo     *argo.Options               `json:"argo,omitempty"`
	Dump     *dump.DumpOptions           `json:"dump,omitempty"`
	Exporter *prometheus.ExporterOptions `json:"exporter,omitempty"`
	Git      *git.Options                `json:"git,omitempty"`
	LogLevel string                      `json:"logLevel,omitempty"`
	Mysql    *database.Options           `json:"mysql,omitempty"`
	Redis    *redis.Options              `json:"redis,omitempty"`
}

func DefaultOptions() *Options {
	return &Options{
		Listen:   ":8080",
		AppStore: helm.NewDefaultOptions(),
		Argo:     argo.NewDefaultArgoOptions(),
		Dump:     dump.NewDefaultDumpOptions(),
		Exporter: prometheus.DefaultExporterOptions(),
		Git:      git.NewDefaultOptions(),
		LogLevel: "debug",
		Mysql:    database.NewDefaultOptions(),
		Redis:    redis.NewDefaultOptions(),
	}
}
