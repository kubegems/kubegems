package log

import (
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const TimeFormat = "2006-01-02 15:04:05.999"

var GlobalLogger, LogrLogger = MustNewLogger()

var AtomicLevel = zap.NewAtomicLevel() // 通过更改 level 可一更改runtime logger的level

func SetLevel(level string) {
	GlobalLogger.Info("logger level updated", zap.String("level", level))
	_ = AtomicLevel.UnmarshalText([]byte(level))
}

func MustNewLogger() (*zap.Logger, logr.Logger) {
	config := zap.NewProductionConfig()
	config.Encoding = "console"
	config.Level = AtomicLevel
	// level from env
	_ = AtomicLevel.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(TimeFormat)
	config.DisableCaller = false    // disable caller
	config.DisableStacktrace = true // disable stacktrace
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	return logger, zapr.NewLogger(logger)
}
