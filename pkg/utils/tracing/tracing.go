package tracing

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/log"
	"github.com/opentracing-contrib/go-gin/ginhttp"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
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

func Init(ctx context.Context) {
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
