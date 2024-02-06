// Copyright 2023 The kubegems.io Authors
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

package otel

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"kubegems.io/kubegems/pkg/log"
	otelgin "kubegems.io/kubegems/pkg/utils/otel/gin"
)

type Options struct {
	Enable       bool   `json:"enable" description:"enable otel"`
	ExcludePaths string `json:"excludePaths" description:"exclude http request paths to sample, split by ','"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Enable:       false,
		ExcludePaths: "/healthz",
	}
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exp),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

func initMeter(ctx context.Context) (*sdkmetric.MeterProvider, error) {
	exp, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, err
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(15*time.Second))))
	otel.SetMeterProvider(mp)
	return mp, nil
}

func Init(ctx context.Context, opts *Options) error {
	if !opts.Enable {
		return nil
	}
	otel.SetLogger(log.LogrLogger)

	if _, err := initTracer(ctx); err != nil {
		return err
	}
	if _, err := initMeter(ctx); err != nil {
		return err
	}

	// start runtime metric
	return runtime.Start(runtime.WithMinimumReadMemStatsInterval(15 * time.Second))
}

// type kubegemsSampler struct{}

// func (as kubegemsSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
// 	result := sdktrace.SamplingResult{
// 		Tracestate: trace.SpanContextFromContext(p.ParentContext).TraceState(),
// 	}
// 	shouldSample := true
// 	for _, att := range p.Attributes {
// 		if att.Key == "kubegems.ignore" && att.Value.AsBool() == true {
// 			shouldSample = false
// 			break
// 		}
// 	}
// 	if shouldSample {
// 		result.Decision = sdktrace.RecordAndSample
// 	} else {
// 		result.Decision = sdktrace.Drop
// 	}
// 	return result
// }

// func (as kubegemsSampler) Description() string {
// 	return "KubegemsSampler"
// }

func PathFilter(opts *Options) otelgin.Filter {
	paths := strings.Split(opts.ExcludePaths, ",")
	return func(c *gin.Context) bool {
		for _, excludePath := range paths {
			if c.Request.URL.Path == excludePath {
				return false
			}
		}
		return true
	}
}

func UseRealPath() otelgin.SpanNameGenerater {
	return func(c *gin.Context) string {
		return c.Request.URL.Path
	}
}
