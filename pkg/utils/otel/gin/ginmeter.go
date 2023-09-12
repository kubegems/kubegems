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

package otelgin

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// Server HTTP metrics.
const (
	RequestCount          = "http.server.request_count"           // Incoming request count total
	RequestContentLength  = "http.server.request_content_length"  // Incoming request bytes total
	ResponseContentLength = "http.server.response_content_length" // Incoming response bytes total
	ServerLatency         = "http.server.duration"                // Incoming end to end duration, microseconds
)

const (
	instrumentationName = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	counters       map[string]metric.Int64Counter
	valueRecorders map[string]metric.Float64Histogram
)

func MeterMiddleware(service string) gin.HandlerFunc {
	counters = make(map[string]metric.Int64Counter)
	valueRecorders = make(map[string]metric.Float64Histogram)
	meter := otel.Meter(instrumentationName)
	requestCounter, err := meter.Int64Counter(RequestCount)
	handleErr(err)
	serverLatencyMeasure, err := meter.Float64Histogram(ServerLatency)
	handleErr(err)

	counters[RequestCount] = requestCounter
	valueRecorders[ServerLatency] = serverLatencyMeasure
	return func(c *gin.Context) {
		requestStartTime := time.Now()
		attributes := metric.WithAttributes(semconv.HTTPServerMetricAttributesFromHTTPRequest(service, c.Request)...)
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		c.Next()
		// Use floating point division here for higher precision (instead of Millisecond method).
		// 由于Bucket分辨率的问题，这里只能记录为millseconds而不是seconds
		elapsedTime := float64(time.Since(requestStartTime)) / float64(time.Millisecond)
		counters[RequestCount].Add(ctx, 1, attributes)
		valueRecorders[ServerLatency].Record(ctx, elapsedTime, attributes)
	}
}

func handleErr(err error) {
	if err != nil {
		otel.Handle(err)
	}
}
