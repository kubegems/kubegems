package options

import (
	microserviceoptions "kubegems.io/pkg/service/handlers/microservice/options"
	"kubegems.io/pkg/utils/agents"
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
	Agent     *agents.Options           `json:"agent"`
	System    *system.Options           `json:"system"`
	Appstore  *helm.Options             `json:"appstore" description:"appstore配置"`
	Argo      *argo.Options             `json:"argo" description:"argo配置"`
	DebugMode bool                      `json:"debugmode" description:"debug模式"`
	Exporter  *exporter.ExporterOptions `json:"exporter" description:"prometheus exporter配置"`
	Git       *git.Options              `json:"git" description:"git配置"`
	JWT       *oauth.JWTOptions         `json:"jwt" description:"jwt配置"`
	LogLevel  string                    `json:"loglevel" description:"日志等级"`
	Msgbus    *msgbus.Options           `json:"msgbus" description:"msgbus 实时消息网关配置"`
	Mysql     *database.Options         `json:"mysql" description:"数据库配置"`
	Redis     *redis.Options            `json:"redis" description:"redis配置"`
}

type OnlineOptions struct {
	Oauth        *oauth.Options                           `json:"oauth" description:"oauth配置"`
	Microservice *microserviceoptions.MicroserviceOptions `json:"microservice" description:"microservice 配置"`
}

func NewOnlineOptions() *OnlineOptions {
	return &OnlineOptions{
		Oauth:        oauth.NewDefaultOptions(),
		Microservice: microserviceoptions.NewDefaultOptions(),
	}
}

func DefaultOptions() *Options {
	defaultoptions := &Options{
		Agent:     agents.NewDefaultOptions(),
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
