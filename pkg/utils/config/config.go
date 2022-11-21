// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"kubegems.io/kubegems/pkg/log"
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

var DefaultValidator = validator.New()

func Validate(data any) error {
	return DefaultValidator.Struct(data)
}

func Print(fs *pflag.FlagSet) {
	fs.VisitAll(func(flag *pflag.Flag) {
		if flag.Changed {
			logConfig("flag", flag.Name, flag.Value.String())
		}
	})
}

func AutoRegisterFlags(fs *pflag.FlagSet, prefix string, data interface{}) {
	node := ParseStruct(data)
	if !node.Value.CanAddr() {
		log.Error(ErrCantRegister, "must be a pointer to a struct", "data", data)
	}
	registerFlagSet(fs, prefix, node.Children)
}

func LoadEnv(fs *pflag.FlagSet) {
	flagNameToEnvKey := func(fname string) string {
		return strings.ToUpper(strings.ReplaceAll(fname, "-", "_"))
	}
	fs.VisitAll(func(f *pflag.Flag) {
		envname := flagNameToEnvKey(f.Name)
		val, ok := os.LookupEnv(envname)
		if ok {
			logConfig("env", envname, val)
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
		log.Warnf("no config file found or config file format error: %v", err)
	}

	fs.VisitAll(func(f *pflag.Flag) {
		filekeyname := flagNameToConfigKey(f.Name)
		val := v.GetString(filekeyname)
		if val != "" {
			logConfig("file", filekeyname, val)
			_ = f.Value.Set(val)
		}
	})
}

func logConfig(from, k, v string) {
	if strings.Contains(strings.ToLower(k), "password") {
		v = strings.Repeat("*", len(v))
	}
	log.Infof("config from %s: %s=%s", from, k, v)
}
