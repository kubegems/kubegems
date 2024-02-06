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

package metrics

import (
	"context"
	"database/sql"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const instrumName = "opentelemetry/otel"

type config struct {
	tracerProvider trace.TracerProvider
	tracer         trace.Tracer

	meterProvider metric.MeterProvider
	meter         metric.Meter

	attrs []metric.ObserveOption
}

func newConfig() *config {
	c := &config{
		tracerProvider: otel.GetTracerProvider(),
		meterProvider:  otel.GetMeterProvider(),
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

	maxOpenConns, _ := meter.Int64ObservableGauge("go.sql.connections_max_open",
		metric.WithDescription("Maximum number of open connections to the database"))
	openConns, _ := meter.Int64ObservableGauge("go.sql.connections_open",
		metric.WithDescription("The number of established connections both in use and idle"))
	inUseConns, _ := meter.Int64ObservableGauge("go.sql.connections_in_use",
		metric.WithDescription("The number of connections currently in use"))
	idleConns, _ := meter.Int64ObservableGauge("go.sql.connections_idle",
		metric.WithDescription("The number of idle connections"))
	connsWaitCount, _ := meter.Int64ObservableCounter("go.sql.connections_wait_count",
		metric.WithDescription("The total number of connections waited for"))
	connsWaitDuration, _ := meter.Int64ObservableCounter("go.sql.connections_wait_duration",
		metric.WithDescription("The total time blocked waiting for a new connection"),
		metric.WithUnit("nanoseconds"))
	connsClosedMaxIdle, _ := meter.Int64ObservableCounter("go.sql.connections_closed_max_idle",
		metric.WithDescription("The total number of connections closed due to SetMaxIdleConns"))
	connsClosedMaxIdleTime, _ := meter.Int64ObservableCounter("go.sql.connections_closed_max_idle_time",
		metric.WithDescription("The total number of connections closed due to SetConnMaxIdleTime"))
	connsClosedMaxLifetime, _ := meter.Int64ObservableCounter("go.sql.connections_closed_max_lifetime",
		metric.WithDescription("The total number of connections closed due to SetConnMaxLifetime"))

	_, err := meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			stats := db.Stats()
			observer.ObserveInt64(maxOpenConns, int64(stats.MaxOpenConnections), labels...)
			observer.ObserveInt64(openConns, int64(stats.OpenConnections), labels...)
			observer.ObserveInt64(inUseConns, int64(stats.InUse), labels...)
			observer.ObserveInt64(idleConns, int64(stats.Idle), labels...)
			observer.ObserveInt64(connsWaitCount, stats.WaitCount, labels...)
			observer.ObserveInt64(connsWaitDuration, int64(stats.WaitDuration), labels...)
			observer.ObserveInt64(connsClosedMaxIdle, stats.MaxIdleClosed, labels...)
			observer.ObserveInt64(connsClosedMaxIdleTime, stats.MaxIdleTimeClosed, labels...)
			observer.ObserveInt64(connsClosedMaxLifetime, stats.MaxLifetimeClosed, labels...)
			return nil
		},
		maxOpenConns, openConns, inUseConns, idleConns,
		connsWaitCount, connsWaitDuration, connsClosedMaxIdle,
		connsClosedMaxIdleTime, connsClosedMaxLifetime,
	)
	if err != nil {
		panic(err)
	}
}
