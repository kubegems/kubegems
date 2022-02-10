package options

import (
	"github.com/spf13/pflag"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/oauth"
	microserviceoptions "kubegems.io/pkg/service/handlers/microservice/options"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/chartmuseum"
	"kubegems.io/pkg/utils/exporter"
	"kubegems.io/pkg/utils/git"
	"kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/msgbus"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/utils/system"
)

type Options struct {
	LogLevel     string                                   `yaml:"loglevel" head_comment:"日志等级"`
	DebugMode    bool                                     `yaml:"debugmode" head_comment:"debug模式开关"`
	TestMode     bool                                     `yaml:"testmode" head_comment:"测试模式"`
	System       *system.SystemOptions                    `yaml:"system" head_comment:"系统配置"`
	Mysql        *models.MySQLOptions                     `yaml:"mysql" head_comment:"数据库配置"`
	Redis        *redis.Options                           `yaml:"redis" head_comment:"redis 缓存配置"`
	Appstore     *chartmuseum.AppstoreOptions             `yaml:"appstore" head_comment:"appstore helm地址配置"`
	Oauth        *oauth.OauthOptions                      `yaml:"oauth" head_comment:"oauth配置"`
	Git          *git.Options                             `yaml:"git" head_comment:"git 配置"`
	Argo         *argo.Options                            `yaml:"argo" head_comment:"argo 相关配置"`
	Exporter     *exporter.ExporterOptions                `yaml:"exporter" head_comment:"prometheus exporter 配置"`
	Msgbus       *msgbus.MsgbusOptions                    `yaml:"msgbus" head_comment:"msgbus 实时消息网关配置"`
	Installer    *kube.InstallerOptions                   `yaml:"installer" head_comment:"installer 集群安装配置"`
	Microservice *microserviceoptions.MicroserviceOptions `yaml:"microservice" head_comment:"microservice 配置"`
}

func DefaultOptions() *Options {
	return &Options{
		DebugMode:    false,
		LogLevel:     "debug",
		System:       system.NewDefaultOptions(),
		Git:          git.NewDefaultOptions(),
		Argo:         argo.NewDefaultArgoOptions(),
		Redis:        redis.NewDefaultOptions(),
		Appstore:     chartmuseum.NewDefaultAppstoreOptions(),
		Oauth:        oauth.NewDefaultOauthOptions(),
		Mysql:        models.NewDefaultMySQLOptions(),
		Exporter:     exporter.DefaultExporterOptions(),
		Msgbus:       msgbus.DefaultMsgbusOptions(),
		Installer:    kube.DefaultInstallerOptions(),
		Microservice: microserviceoptions.NewDefaultOptions(),
	}
}

func (o *Options) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.BoolVar(&o.DebugMode, utils.JoinFlagName(prefix, "debugmode"), o.DebugMode, "enable debud mode")
	fs.StringVar(&o.LogLevel, utils.JoinFlagName(prefix, "loglevel"), o.LogLevel, "log level")

	o.System.RegistFlags("system", fs)
	o.Mysql.RegistFlags("mysql", fs)
	o.Redis.RegistFlags("redis", fs)
	o.Appstore.RegistFlags("appstore", fs)
	o.Oauth.RegistFlags("oauth", fs)
	o.Git.RegistFlags("git", fs)
	o.Argo.RegistFlags("argo", fs)
	o.Exporter.RegistFlags("exporter", fs)
	o.Msgbus.RegistFlags("msgbus", fs)
	o.Installer.RegistFlags("installer", fs)
	o.Microservice.RegistFlags("microservice", fs)
}
