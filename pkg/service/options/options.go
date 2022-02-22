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
	LogLevel     string                                   `json:"loglevel" description:"日志等级"`
	DebugMode    bool                                     `json:"debugmode" description:"debug模式"`
	System       *system.Options                          `json:"system" description:"系统配置"`
	Mysql        *database.Options                        `json:"mysql" description:"数据库配置"`
	Redis        *redis.Options                           `json:"redis" description:"redis配置"`
	Appstore     *helm.Options                            `json:"appstore" description:"appstore配置"`
	Oauth        *oauth.Options                           `json:"oauth" description:"oauth配置"`
	Git          *git.Options                             `json:"git" description:"git配置"`
	Argo         *argo.Options                            `json:"argo" description:"argo配置"`
	Exporter     *exporter.ExporterOptions                `json:"exporter" description:"prometheus exporter配置"`
	Msgbus       *msgbus.Options                          `json:"msgbus" description:"msgbus 实时消息网关配置"`
	Microservice *microserviceoptions.MicroserviceOptions `json:"microservice" description:"microservice 配置"`
}

func DefaultOptions() *Options {
	return &Options{
		DebugMode:    false,
		LogLevel:     "debug",
		System:       system.NewDefaultOptions(),
		Git:          git.NewDefaultOptions(),
		Argo:         argo.NewDefaultArgoOptions(),
		Redis:        redis.NewDefaultOptions(),
		Appstore:     helm.NewDefaultOptions(),
		Oauth:        oauth.NewDefaultOauthOptions(),
		Mysql:        database.NewDefaultMySQLOptions(),
		Exporter:     exporter.DefaultExporterOptions(),
		Msgbus:       msgbus.DefaultMsgbusOptions(),
		Microservice: microserviceoptions.NewDefaultOptions(),
	}
}
