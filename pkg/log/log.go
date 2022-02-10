package log

import (
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const timeFormat = "2006-01-02 15:04:05.999"

var GlobalLogger *zap.Logger = zap.NewNop() // init with noop

func IsDebug() bool {
	b, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	return b
}

func LogLevel() string {
	l := os.Getenv("LOG_LEVEL")
	if l == "" {
		return "debug"
	}
	return l
}

func init() {
	_ = Update(IsDebug(), LogLevel())
}

func UpdateGlobalLogger(logger *zap.Logger) {
	GlobalLogger = logger
}

func Update(debug bool, level string) error {
	zapLogger, err := FromBaseZapLogger(debug, level)
	if err != nil {
		return err
	}

	zapLogger.Info("debug level", zap.String("level", level))
	zapLogger.Info("debug enabled", zap.Bool("debug", debug))

	LogrLogger = zapr.NewLogger(zapLogger)

	GlobalLogger = zapLogger
	SetGinDebugPrintRouteFunc()
	return nil
}

func FromBaseZapLogger(debug bool, level string, options ...zap.Option) (*zap.Logger, error) {
	var config zap.Config

	if debug {
		// debug only
		config = zap.NewDevelopmentConfig()
		config.DisableStacktrace = true
		config.EncoderConfig.EncodeLevel = FirstLetterLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(timeFormat)

	} else {
		config = zap.NewProductionConfig()
		config.DisableCaller = true // 生产中是不需要 caller 的
	}
	_ = config.Level.UnmarshalText([]byte(level))
	config.EncoderConfig.ConsoleSeparator = " "
	return config.Build(options...)
}

func FirstLetterLevelEncoder(l zapcore.Level, pae zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.FatalLevel:
		pae.AppendString("[F]")
	case zapcore.ErrorLevel:
		pae.AppendString("[E]")
	case zapcore.WarnLevel:
		pae.AppendString("[W]")
	case zapcore.InfoLevel:
		pae.AppendString("[I]")
	case zapcore.DebugLevel:
		pae.AppendString("[D]")
	default:
		pae.AppendString(l.CapitalString())
	}
}

// GinLoggerMideare is the gin logger handler
func DefaultGinLoggerMideare() gin.HandlerFunc {
	logger := GlobalLogger.WithOptions(zap.AddCallerSkip(1))
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		statusCode := c.Writer.Status()

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("code", statusCode),
			zap.Duration("latency", latency),
		}

		if len(c.Errors) != 0 {
			logger.Error(c.Errors.String(), fields...)
			return
		}
		if statusCode >= http.StatusInternalServerError {
			logger.Error(http.StatusText(statusCode), fields...)
			return
		}
		if statusCode >= http.StatusBadRequest {
			logger.Warn(http.StatusText(statusCode), fields...)
			return
		}
		logger.Debug(http.StatusText(statusCode), fields...)
	}
}

func NewZapLogger(level string, debug bool) (*zap.Logger, error) {
	zapLogger, err := FromBaseZapLogger(debug, level)
	if err != nil {
		return nil, err
	}
	zapLogger.Info("set logger level", zap.String("level", level))
	zapLogger.Info("debug enabled", zap.Bool("debug", debug))
	return zapLogger, nil
}

// Gindebug
func SetGinDebugPrintRouteFunc() {
	logger := GlobalLogger.WithOptions(zap.AddCallerSkip(2))
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		handlerName = path.Base(handlerName)
		fields := []zap.Field{
			zap.String("method", httpMethod),
			zap.String("path", absolutePath),
			zap.String("handler", handlerName),
			zap.Int("count", nuHandlers),
		}
		logger.Info("registered", fields...)
	}
}
