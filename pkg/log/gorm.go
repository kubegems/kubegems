package log

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type GormLogger struct {
	logger                *zap.Logger
	SourceField           string
	SlowThreshold         time.Duration
	SkipErrRecordNotFound bool
}

func NewGormLogger() *GormLogger {
	return &GormLogger{
		logger:                GlobalLogger.WithOptions(zap.AddCallerSkip(3)),
		SlowThreshold:         300 * time.Millisecond,
		SkipErrRecordNotFound: false,
	}
}

func NewGormZapLogger(logger *zap.Logger) *GormLogger {
	return &GormLogger{
		logger:                logger,
		SourceField:           "source",
		SlowThreshold:         300 * time.Millisecond,
		SkipErrRecordNotFound: false,
	}
}

func (l *GormLogger) LogMode(loglevel logger.LogLevel) logger.Interface {
	// all level share a logger
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
	sql, _ := fc()

	fields := []zap.Field{
		zap.String("sql", sql),
		zap.String("latency", latency.String()),
	}
	if l.SourceField != "" {
		fields = append(fields, zap.String(l.SourceField, utils.FileWithLineNum()))
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
