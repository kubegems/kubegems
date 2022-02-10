package prometheus

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/kubegems/gems/pkg/utils/exporter"
)

func RunExporter(ctx context.Context, opts *exporter.ExporterOptions, handler *exporter.Handler) error {
	mu := http.NewServeMux()
	mu.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<head><title>Gemcloud Exporter</title></head>
			<body>
			<h1>Gemcloud Exporter</h1>
			<p><a href="` + opts.MetricPath + `">Metrics</a></p>
			</body>
			</html>`))
	})
	mu.Handle(opts.MetricPath, handler)

	server := &http.Server{Addr: opts.Listen, Handler: mu}
	go func() {
		<-ctx.Done()
		server.Close()
	}()
	logr.FromContextOrDiscard(ctx).Info("prometheus exporter listen on", "address", opts.Listen)
	return server.ListenAndServe()
}
