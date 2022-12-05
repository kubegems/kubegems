package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/metric"
	"kubegems.io/kubegems/pkg/log"
)

func InitRuntimeOtel(ctx context.Context) (shutdownfunc, error) {
	exp, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, err
	}

	// Register the exporter with an SDK via a periodic reader.
	reader := metric.NewPeriodicReader(exp, metric.WithInterval(5*time.Second))
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	global.SetMeterProvider(provider)

	log.Info("Starting runtime instrumentation...")
	return provider.Shutdown, runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
}
