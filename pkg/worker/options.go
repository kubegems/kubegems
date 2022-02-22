package worker

import (
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/exporter"
	"kubegems.io/pkg/utils/git"
	"kubegems.io/pkg/utils/helm"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/worker/dump"
)

type Options struct {
	AppStore *helm.Options             `json:"appStore,omitempty"`
	Argo     *argo.Options             `json:"argo,omitempty"`
	Dump     *dump.DumpOptions         `json:"dump,omitempty"`
	Exporter *exporter.ExporterOptions `json:"exporter,omitempty"`
	Git      *git.Options              `json:"git,omitempty"`
	LogLevel string                    `json:"logLevel,omitempty"`
	Mysql    *database.Options         `json:"mysql,omitempty"`
	Redis    *redis.Options            `json:"redis,omitempty"`
}

func DefaultOptions() *Options {
	return &Options{
		AppStore: helm.NewDefaultOptions(),
		Argo:     argo.NewDefaultArgoOptions(),
		Dump:     dump.NewDefaultDumpOptions(),
		Exporter: exporter.DefaultExporterOptions(),
		Git:      git.NewDefaultOptions(),
		LogLevel: "debug",
		Mysql:    database.NewDefaultOptions(),
		Redis:    redis.NewDefaultOptions(),
	}
}
