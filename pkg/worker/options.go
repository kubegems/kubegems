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
