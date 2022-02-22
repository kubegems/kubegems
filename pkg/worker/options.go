package worker

import (
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/exporter"
	"kubegems.io/pkg/utils/git"
	"kubegems.io/pkg/utils/helm"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/utils/system"
	"kubegems.io/pkg/worker/dump"
)

type Options struct {
	Mysql     *database.Options         `json:"mysql,omitempty"`
	Exporter  *exporter.ExporterOptions `json:"exporter,omitempty"`
	Argo      *argo.Options             `json:"argo,omitempty"`
	AppStore  *helm.Options             `json:"appStore,omitempty"`
	Dump      *dump.DumpOptions         `json:"dump,omitempty"`
	System    *system.Options           `json:"system,omitempty"`
	DebugMode bool                      `json:"debugMode,omitempty"`
	LogLevel  string                    `json:"logLevel,omitempty"`
	Redis     *redis.Options            `json:"redis,omitempty"`
	Git       *git.Options              `json:"git,omitempty"`
}

func DefaultOptions() *Options {
	return &Options{
		Mysql:     database.NewDefaultMySQLOptions(),
		Exporter:  exporter.DefaultExporterOptions(),
		Argo:      argo.NewDefaultArgoOptions(),
		AppStore:  helm.NewDefaultOptions(),
		Dump:      dump.NewDefaultDumpOptions(),
		System:    system.NewDefaultOptions(),
		DebugMode: false,
		Redis:     redis.NewDefaultOptions(),
		Git:       git.NewDefaultOptions(),
		LogLevel:  "debug",
	}
}
