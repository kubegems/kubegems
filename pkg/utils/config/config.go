package config

import (
	"os"
	"strings"

	"github.com/kubegems/gems/pkg/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Parse 从多个方式加载配置
/*
 * 配置文件加载有如下优先级：
 1. 命令行参数
 2. 环境变量
 3. 配置文件
 4. 默认值

- 高优先级的配置若存在，会覆盖低优先级已存在的配置
- 若所有配置均不存在，则使用默认值

对于需要做配置的项目，需要先设置 flag，环境和配置文件会使用已经设置flag进行配置

举例：
若需要增加配置项目,需要配置使用的结构并设置默认值，例：Foo{Bar:"默认值"},
然后使用 pflag 配置命令行参数：

	fs.StringVarP(&options.Foo.Bar, "foo-bar", "", options.Foo.Bar, "foo bar")

配置完成后,Parse 会根据 plagset 中已有配置 "foo-bar",获取对应的环境变量 "FOO_BAR"，以及对应的配置文件项 "foo.bar"
*/
func Parse(fs *pflag.FlagSet) error {
	// 从默认值配置
	// fs 中已有默认值
	// 从文件配置
	LoadConfigFile(fs)
	// 从环境变量配置
	LoadEnv(fs)
	// 从命令行配置
	if err := fs.Parse(os.Args); err != nil {
		return err
	}
	// print
	Print(fs)
	return nil
}

func Print(fs *pflag.FlagSet) {
	fs.VisitAll(func(flag *pflag.Flag) {
		if flag.Changed {
			log.Infof("config from flag: --%s=%s", flag.Name, flag.Value)
		}
	})
}

func LoadEnv(fs *pflag.FlagSet) {
	flagNameToEnvKey := func(fname string) string {
		return strings.ToUpper(strings.ReplaceAll(fname, "-", "_"))
	}
	fs.VisitAll(func(f *pflag.Flag) {
		envname := flagNameToEnvKey(f.Name)
		val, ok := os.LookupEnv(envname)
		if ok {
			log.Infof("config from env: %s=%s", envname, val)
			_ = f.Value.Set(val)
		}
	})
}

func LoadConfigFile(fs *pflag.FlagSet) {
	flagNameToConfigKey := func(fname string) string {
		return strings.ToLower(strings.ReplaceAll(fname, "-", "."))
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("config")
	if err := v.ReadInConfig(); err != nil {
		log.Warnf("no config file found")
	}

	fs.VisitAll(func(f *pflag.Flag) {
		filekeyname := flagNameToConfigKey(f.Name)
		val := v.GetString(filekeyname)
		if val != "" {
			log.Infof("config from file: %s=%s", filekeyname, val)
			_ = f.Value.Set(val)
		}
	})
}
