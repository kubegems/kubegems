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
	"context"
	"errors"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

const SlowQueryThreshold = 300 * time.Millisecond

func NewDefaultGormZapLogger() *GormLogger {
	return NewGormZapLogger(GlobalLogger)
}

func NewGormZapLogger(logger *zap.Logger) *GormLogger {
	logger = logger.WithOptions(zap.AddCallerSkip(1))
	return &GormLogger{
		logger:                logger,
		SourceField:           "source",
		SlowThreshold:         SlowQueryThreshold,
		SkipErrRecordNotFound: true,
	}
}

var _ logger.Interface = &GormLogger{}

type GormLogger struct {
	logger                *zap.Logger
	SourceField           string
	SlowThreshold         time.Duration
	SkipErrRecordNotFound bool
}

func (l *GormLogger) LogMode(loglevel logger.LogLevel) logger.Interface {
	return l
}

func (l *GormLogger) Info(ctx context.Context, s string, args ...interface{}) {
	l.logger.Sugar().Infof(s, args...)
}

func (l *GormLogger) Warn(ctx context.Context, s string, args ...interface{}) {
	l.logger.Sugar().Warnf(s, args...)
}

func (l *GormLogger) Error(ctx context.Context, s string, args ...interface{}) {
	l.logger.Sugar().Errorf(s, args...)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	latency := time.Since(begin)
	sql, effect := fc()

	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Int64("rows", effect),
		zap.String("latency", latency.String()),
	}
	if l.SourceField != "" {
		fields = append(fields, zap.String(l.SourceField, filepath.Base(utils.FileWithLineNum())))
	}
	if err != nil && !(errors.Is(err, gorm.ErrRecordNotFound) && l.SkipErrRecordNotFound) {
		l.logger.Error(err.Error(), fields...)
		return
	}
	if l.SlowThreshold != 0 && latency > l.SlowThreshold {
		l.logger.Warn("slow query", fields...)
		return
	}
	l.logger.Debug("success", fields...)
}
