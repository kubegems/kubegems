package worker

import (
	"github.com/spf13/pflag"
	"kubegems.io/pkg/utils"
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
	Mysql     *database.MySQLOptions    `yaml:"mysql"`
	Exporter  *exporter.ExporterOptions `yaml:"exporter"`
	Argo      *argo.Options             `yaml:"argo"`
	Appstore  *helm.Options             `yaml:"appstore"`
	Dump      *dump.DumpOptions         `yaml:"dump"`
	System    *system.SystemOptions     `yaml:"system" head_comment:"系统配置"`
	DebugMode bool                      `yaml:"debugmode"` // enable debug mode
	LogLevel  string                    `yaml:"loglevel"`
	Redis     *redis.Options            `yaml:"redis" head_comment:"redis 缓存配置"`
	Git       *git.Options              `yaml:"git" head_comment:"git server"`
}

func DefaultOptions() *Options {
	return &Options{
		Mysql:     database.NewDefaultMySQLOptions(),
		Exporter:  exporter.DefaultExporterOptions(),
		Argo:      argo.NewDefaultArgoOptions(),
		Appstore:  helm.NewDefaultOptions(),
		Dump:      dump.NewDefaultDumpOptions(),
		System:    system.NewDefaultOptions(),
		DebugMode: false,
		Redis:     redis.NewDefaultOptions(),
		Git:       git.NewDefaultOptions(),
		LogLevel:  "debug",
	}
}

func (o *Options) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.BoolVar(&o.DebugMode, utils.JoinFlagName(prefix, "debugmode"), o.DebugMode, "enable debud mode")
	fs.StringVar(&o.LogLevel, utils.JoinFlagName(prefix, "loglevel"), o.LogLevel, "log level")
	o.Mysql.RegistFlags("mysql", fs)
	o.Exporter.RegistFlags("exporter", fs)
	o.Argo.RegistFlags("argo", fs)
	o.Appstore.RegistFlags("appstore", fs)
	o.Dump.RegistFlags("dump", fs)
	o.System.RegistFlags("system", fs)
	o.Redis.RegistFlags("redis", fs)
	o.Git.RegistFlags("git", fs)
}
