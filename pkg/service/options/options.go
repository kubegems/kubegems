package options

import (
	microserviceoptions "kubegems.io/pkg/service/handlers/microservice/options"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/exporter"
	"kubegems.io/pkg/utils/git"
	"kubegems.io/pkg/utils/helm"
	"kubegems.io/pkg/utils/msgbus"
	"kubegems.io/pkg/utils/oauth"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/utils/system"
)

type Options struct {
	System    *system.Options           `json:"system,omitempty"`
	Appstore  *helm.Options             `json:"appstore,omitempty"`
	Argo      *argo.Options             `json:"argo,omitempty"`
	DebugMode bool                      `json:"debugMode,omitempty"`
	Exporter  *exporter.ExporterOptions `json:"exporter,omitempty"`
	Git       *git.Options              `json:"git,omitempty"`
	JWT       *oauth.JWTOptions         `json:"jwt,omitempty"`
	LogLevel  string                    `json:"logLevel,omitempty"`
	Msgbus    *msgbus.Options           `json:"msgbus,omitempty"`
	Mysql     *database.Options         `json:"mysql,omitempty"`
	Redis     *redis.Options            `json:"redis,omitempty"`
}

type OnlineOptions struct {
	Oauth        *oauth.Options                           `json:"oauth,omitempty"`
	Microservice *microserviceoptions.MicroserviceOptions `json:"microservice,omitempty"`
}

func NewOnlineOptions() *OnlineOptions {
	return &OnlineOptions{
		Oauth:        oauth.NewDefaultOptions(),
		Microservice: microserviceoptions.NewDefaultOptions(),
	}
}

func DefaultOptions() *Options {
	defaultoptions := &Options{
		Appstore:  helm.NewDefaultOptions(),
		Argo:      argo.NewDefaultArgoOptions(),
		DebugMode: false,
		Exporter:  exporter.DefaultExporterOptions(),
		Git:       git.NewDefaultOptions(),
		JWT:       oauth.NewDefaultJWTOptions(),
		LogLevel:  "debug",
		Msgbus:    msgbus.DefaultMsgbusOptions(),
		Mysql:     database.NewDefaultMySQLOptions(),
		Redis:     redis.NewDefaultOptions(),
		System:    system.NewDefaultOptions(),
	}
	defaultoptions.System.Listen = ":8020"
	return defaultoptions
}
