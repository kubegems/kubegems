package metrics

import (
	"context"
	"database/sql"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/trace"
)

const instrumName = "opentelemetry/otel"

type config struct {
	tracerProvider trace.TracerProvider
	tracer         trace.Tracer

	meterProvider metric.MeterProvider
	meter         metric.Meter

	attrs []attribute.KeyValue
}

func newConfig() *config {
	c := &config{
		tracerProvider: otel.GetTracerProvider(),
		meterProvider:  global.MeterProvider(),
		tracer:         nil,
		meter:          nil,
		attrs:          nil,
	}
	return c
}

// ReportDBStatsMetrics reports DBStats metrics using OpenTelemetry Metrics API.
func ReportDBStatsMetrics(db *sql.DB) {
	cfg := newConfig()

	if cfg.meter == nil {
		cfg.meter = cfg.meterProvider.Meter(instrumName)
	}

	meter := cfg.meter
	labels := cfg.attrs

	maxOpenConns, _ := meter.AsyncInt64().Gauge(
		"go.sql.connections_max_open",
		instrument.WithDescription("Maximum number of open connections to the database"),
	)
	openConns, _ := meter.AsyncInt64().Gauge(
		"go.sql.connections_open",
		instrument.WithDescription("The number of established connections both in use and idle"),
	)
	inUseConns, _ := meter.AsyncInt64().Gauge(
		"go.sql.connections_in_use",
		instrument.WithDescription("The number of connections currently in use"),
	)
	idleConns, _ := meter.AsyncInt64().Gauge(
		"go.sql.connections_idle",
		instrument.WithDescription("The number of idle connections"),
	)
	connsWaitCount, _ := meter.AsyncInt64().Counter(
		"go.sql.connections_wait_count",
		instrument.WithDescription("The total number of connections waited for"),
	)
	connsWaitDuration, _ := meter.AsyncInt64().Counter(
		"go.sql.connections_wait_duration",
		instrument.WithDescription("The total time blocked waiting for a new connection"),
		instrument.WithUnit("nanoseconds"),
	)
	connsClosedMaxIdle, _ := meter.AsyncInt64().Counter(
		"go.sql.connections_closed_max_idle",
		instrument.WithDescription("The total number of connections closed due to SetMaxIdleConns"),
	)
	connsClosedMaxIdleTime, _ := meter.AsyncInt64().Counter(
		"go.sql.connections_closed_max_idle_time",
		instrument.WithDescription("The total number of connections closed due to SetConnMaxIdleTime"),
	)
	connsClosedMaxLifetime, _ := meter.AsyncInt64().Counter(
		"go.sql.connections_closed_max_lifetime",
		instrument.WithDescription("The total number of connections closed due to SetConnMaxLifetime"),
	)

	if err := meter.RegisterCallback(
		[]instrument.Asynchronous{
			maxOpenConns,

			openConns,
			inUseConns,
			idleConns,

			connsWaitCount,
			connsWaitDuration,
			connsClosedMaxIdle,
			connsClosedMaxIdleTime,
			connsClosedMaxLifetime,
		},
		func(ctx context.Context) {
			stats := db.Stats()

			maxOpenConns.Observe(ctx, int64(stats.MaxOpenConnections), labels...)

			openConns.Observe(ctx, int64(stats.OpenConnections), labels...)
			inUseConns.Observe(ctx, int64(stats.InUse), labels...)
			idleConns.Observe(ctx, int64(stats.Idle), labels...)

			connsWaitCount.Observe(ctx, stats.WaitCount, labels...)
			connsWaitDuration.Observe(ctx, int64(stats.WaitDuration), labels...)
			connsClosedMaxIdle.Observe(ctx, stats.MaxIdleClosed, labels...)
			connsClosedMaxIdleTime.Observe(ctx, stats.MaxIdleTimeClosed, labels...)
			connsClosedMaxLifetime.Observe(ctx, stats.MaxLifetimeClosed, labels...)
		},
	); err != nil {
		panic(err)
	}
}
