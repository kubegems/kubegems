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
	"net/http"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Gindebug
func SetGinDebugPrintRouteFunc(logger *zap.Logger) {
	const callerSkip = 2
	logger = logger.WithOptions(zap.AddCallerSkip(callerSkip))
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

func DefaultGinLoggerMideare() gin.HandlerFunc {
	return NewGinLoggerMideare(GlobalLogger)
}

// GinLoggerMideare is the gin logger handler
func NewGinLoggerMideare(logger *zap.Logger) gin.HandlerFunc {
	logger = logger.WithOptions(zap.AddCallerSkip(1))
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
		logger.Info(http.StatusText(statusCode), fields...)
	}
}
