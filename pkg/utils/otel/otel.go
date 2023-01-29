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
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"kubegems.io/kubegems/pkg/log"
)

type Options struct {
	Enable bool `json:"enable" description:"enable otel"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Enable: false,
	}
}

func Init(ctx context.Context, opts *Options) error {
	if !opts.Enable {
		return nil
	}
	otel.SetLogger(log.LogrLogger)

	// trace
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return err
	}
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(traceExporter),
	)
	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// metric
	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return err
	}
	reader := metric.NewPeriodicReader(metricExporter, metric.WithInterval(5*time.Second))
	metricProvider := metric.NewMeterProvider(metric.WithReader(reader))
	global.SetMeterProvider(metricProvider)

	// start runtime metric
	return runtime.Start(runtime.WithMinimumReadMemStatsInterval(15 * time.Second))
}
