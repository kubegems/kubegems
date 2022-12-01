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

package exporter

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	promcollectors "github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"go.uber.org/zap"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils"
	gempro "kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/system"
)

var (
	MetricPath             = utils.StrOrDef(os.Getenv("METRIC_PATH"), "/metrics")
	IncludeExporterMetrics = false
	MaxRequests            = 40
)

// Handler wraps an unfiltered http.Handler but uses a filtered Handler,
// created on the fly, if filtering is requested. Create instances with
// newHandler.
type Handler struct {
	unfilteredHandler http.Handler
	// exporterMetricsRegistry is a separate registry for the metrics about
	// the exporter itself.
	exporterMetricsRegistry *prometheus.Registry
	includeExporterMetrics  bool
	maxRequests             int
	logger                  *log.Logger
}

func NewHandler(namespace string, collectors map[string]Collectorfunc) *Handler {
	setNamespace(namespace)
	for k, v := range collectors {
		registerCollector(k, v)
	}

	return newHandlerWith(IncludeExporterMetrics, MaxRequests, log.GlobalLogger.Sugar())
}

func (h *Handler) Run(ctx context.Context, opts *gempro.ExporterOptions) error {
	mu := http.NewServeMux()
	mu.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<head><title>Gemcloud Exporter</title></head>
			<body>
			<h1>Gemcloud Exporter</h1>
			<p><a href="` + MetricPath + `">Metrics</a></p>
			</body>
			</html>`))
	})
	mu.Handle(MetricPath, h)

	log.FromContextOrDiscard(ctx).Info("prometheus exporter listen on", "address", opts.Listen)
	return system.ListenAndServeContext(ctx, opts.Listen, nil, mu)
}

func newHandlerWith(includeExporterMetrics bool, maxRequests int, logger *log.Logger) *Handler {
	h := &Handler{
		exporterMetricsRegistry: prometheus.NewRegistry(),
		includeExporterMetrics:  includeExporterMetrics,
		maxRequests:             maxRequests,
		logger:                  logger,
	}
	if h.includeExporterMetrics {
		h.exporterMetricsRegistry.MustRegister(
			promcollectors.NewProcessCollector(promcollectors.ProcessCollectorOpts{}),
			promcollectors.NewGoCollector(),
		)
	}
	if innerHandler, err := h.innerHandler(); err != nil {
		panic(fmt.Sprintf("Couldn't create metrics handler: %s", err))
	} else {
		h.unfilteredHandler = innerHandler
	}
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["collect[]"]
	h.logger.Debug("filters ", filters)

	if len(filters) == 0 {
		// No filters, use the prepared unfiltered handler.
		h.unfilteredHandler.ServeHTTP(w, r)
		return
	}
	// To serve filtered metrics, we create a filtering handler on the fly.
	filteredHandler, err := h.innerHandler(filters...)
	if err != nil {
		h.logger.Warn("Couldn't create filtered metrics handler: ", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Couldn't create filtered metrics handler: %s", err)
		return
	}
	filteredHandler.ServeHTTP(w, r)
}

// innerHandler is used to create both the one unfiltered http.Handler to be
// wrapped by the outer handler and also the filtered handlers created on the
// fly. The former is accomplished by calling innerHandler without any arguments
// (in which case it will log all the collectors enabled via command-line
// flags).
func (h *Handler) innerHandler(filters ...string) (http.Handler, error) {
	nc, err := newBaseCollector(h.logger, filters...)
	if err != nil {
		return nil, fmt.Errorf("couldn't create collector: %s", err)
	}

	// Only log the creation of an unfiltered handler, which should happen
	// only once upon startup.
	if len(filters) == 0 {
		h.logger.Info("Enabled collectors")
		collectors := []string{}
		for n := range nc.Collectors {
			collectors = append(collectors, n)
		}
		sort.Strings(collectors)
		for _, c := range collectors {
			h.logger.Info("collector ", c)
		}
	}

	r := prometheus.NewRegistry()
	r.MustRegister(version.NewCollector(getNamespace()))
	if err := r.Register(nc); err != nil {
		return nil, fmt.Errorf("couldn't register node collector: %s", err)
	}
	handler := promhttp.HandlerFor(
		prometheus.Gatherers{h.exporterMetricsRegistry, r},
		promhttp.HandlerOpts{
			ErrorLog:            zap.NewStdLog(h.logger.Desugar()), // todo: remove zap import
			ErrorHandling:       promhttp.ContinueOnError,
			MaxRequestsInFlight: h.maxRequests,
			Registry:            h.exporterMetricsRegistry,
		},
	)
	if h.includeExporterMetrics {
		// Note that we have to use h.exporterMetricsRegistry here to
		// use the same promhttp metrics for all expositions.
		handler = promhttp.InstrumentMetricHandler(
			h.exporterMetricsRegistry, handler,
		)
	}
	return handler, nil
}
