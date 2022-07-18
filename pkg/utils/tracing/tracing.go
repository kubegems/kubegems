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

package tracing

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opentracing-contrib/go-gin/ginhttp"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
	"kubegems.io/kubegems/pkg/log"
)

type logger struct{}

// Error logs a message at error priority
func (logger) Error(msg string) {
	log.WithField("jaeger", "tracing").Error(msg)
}

// Infof logs a message at info priority
func (logger) Infof(msg string, args ...interface{}) {
	log.WithField("jaeger", "tracing").Infof(msg, args...)
}

func SetGlobal(ctx context.Context) {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Warnf("could not parse jaeger env vars: %s", err.Error())
		return
	}

	tracer, closer, err := cfg.NewTracer(
		config.Logger(logger{}),
	)
	if err != nil {
		log.Warnf("could not initialize jaeger tracer: %s", err.Error())
		return
	}

	go func() {
		<-ctx.Done()
		closer.Close()
	}()

	opentracing.SetGlobalTracer(tracer)
}

func GinMiddleware() gin.HandlerFunc {
	return ginhttp.Middleware(opentracing.GlobalTracer(),
		ginhttp.OperationNameFunc(func(r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)
}
