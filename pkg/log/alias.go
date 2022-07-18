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

package log

import (
	"github.com/go-logr/logr"
	"go.uber.org/zap"
)

// 背景： https://github.com/go-logr/logr#background
// 总体上有几条：

// 1. 日志需要结构化，在云原生的环境下，结构化的日志相比于 format 日志更机器可读，可统计，可搜索，信息要素完善。
// 1. 日志最终仅分为错误和没有错误两类。对应 .Info .Error 函数。（警告级别的日志没有人会关心，所以根本没用）see: https://dave.cheney.net/2015/11/05/lets-talk-about-logging
// 1. 对于日志,除了用debug trace warn 等区别等级 ，还可以用更灵活的 .V() 设置等级，可以自行判断根据日志重要性设置。

// 对于 caller，如果logger使用的正确，是不需要caller的 使用 .WithName() 可以手动区分logger所在模块，且使用caller stacktrace 会增加额外的开销。

var NewContext = logr.NewContext

var FromContextOrDiscard = logr.FromContextOrDiscard

func Error(err error, msg string, keysAndValues ...interface{}) {
	LogrLogger.WithCallDepth(1).Error(err, msg, keysAndValues...)
}

func Info(msg string, keysAndValues ...interface{}) {
	LogrLogger.WithCallDepth(1).Info(msg, keysAndValues...)
}

func V(level int) logr.Logger {
	return LogrLogger.V(level)
}

func WithName(name string) logr.Logger {
	return LogrLogger.WithName(name)
}

func WithValues(keysAndValues ...interface{}) logr.Logger {
	return LogrLogger.WithValues(keysAndValues...)
}

type (
	Logger = zap.SugaredLogger
)

func Fatalf(fmt string, v ...interface{}) {
	GlobalLogger.WithOptions(zap.AddCallerSkip(1)).Sugar().Fatalf(fmt, v...)
}

func Errorf(fmt string, v ...interface{}) {
	GlobalLogger.WithOptions(zap.AddCallerSkip(1)).Sugar().Errorf(fmt, v...)
}

func Warnf(fmt string, v ...interface{}) {
	GlobalLogger.WithOptions(zap.AddCallerSkip(1)).Sugar().Warnf(fmt, v...)
}

func Infof(fmt string, v ...interface{}) {
	GlobalLogger.WithOptions(zap.AddCallerSkip(1)).Sugar().Infof(fmt, v...)
}

func Debugf(fmt string, v ...interface{}) {
	GlobalLogger.WithOptions(zap.AddCallerSkip(1)).Sugar().Debugf(fmt, v...)
}

func Tracef(fmt string, v ...interface{}) {
	GlobalLogger.WithOptions(zap.AddCallerSkip(1)).Sugar().Debugf(fmt, v...)
}

// use .Info("",fields), or .With(fields).Info("") instead
func WithField(key string, value interface{}) *zap.SugaredLogger {
	return GlobalLogger.Sugar().With(key, value)
}
