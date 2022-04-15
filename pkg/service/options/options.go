package options

import (
	microservice "kubegems.io/pkg/service/handlers/microservice/options"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/git"
	"kubegems.io/pkg/utils/helm"
	"kubegems.io/pkg/utils/jwt"
	"kubegems.io/pkg/utils/msgbus"
	"kubegems.io/pkg/utils/prometheus"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/utils/system"
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
	}
	defaultoptions.System.Listen = ":8020"
	return defaultoptions
}
