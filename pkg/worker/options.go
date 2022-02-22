package worker

import (
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/exporter"
	"kubegems.io/pkg/utils/git"
	"kubegems.io/pkg/utils/helm"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/worker/dump"
)

type Options struct {
	Agent     *agents.Options           `json:"agent,omitempty"`
	AppStore  *helm.Options             `json:"appStore,omitempty"`
	Argo      *argo.Options             `json:"argo,omitempty"`
	DebugMode bool                      `json:"debugMode,omitempty"`
	Dump      *dump.DumpOptions         `json:"dump,omitempty"`
	Exporter  *exporter.ExporterOptions `json:"exporter,omitempty"`
	Git       *git.Options              `json:"git,omitempty"`
	LogLevel  string                    `json:"logLevel,omitempty"`
	Mysql     *database.Options         `json:"mysql,omitempty"`
	Redis     *redis.Options            `json:"redis,omitempty"`
}

func DefaultOptions() *Options {
	return &Options{
		Agent:     agents.NewDefaultOptions(),
		AppStore:  helm.NewDefaultOptions(),
		Argo:      argo.NewDefaultArgoOptions(),
		DebugMode: false,
		Dump:      dump.NewDefaultDumpOptions(),
		Exporter:  exporter.DefaultExporterOptions(),
		Git:       git.NewDefaultOptions(),
		LogLevel:  "debug",
		Mysql:     database.NewDefaultMySQLOptions(),
		Redis:     redis.NewDefaultOptions(),
	}
}
