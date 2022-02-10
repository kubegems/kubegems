package log

import (
	"path"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
)

var LogrLogger logr.Logger = logr.Discard() // init with discard

func NewLogger(level string, debug bool) (logr.Logger, error) {
	zapLogger, err := NewZapLogger(level, debug)
	if err != nil {
		return logr.Discard(), err
	}
	return zapr.NewLogger(zapLogger), nil
}

func SetGinDebugPrintRouteFuncLogger(logger logr.Logger) {
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		handlerName = path.Base(handlerName)
		fields := []interface{}{
			"method", httpMethod,
			"path", absolutePath,
			"handler", handlerName,
			"count", nuHandlers,
		}
		logger.Info("registered", fields...)
	}
}
