package system

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func ListenAndServeContextGRPCAndHTTP(ctx context.Context, listen string, tls *tls.Config, httphandler http.Handler, grpchandler http.Handler) error {
	log := logr.FromContextOrDiscard(ctx)
	if grpchandler != nil {
		httphandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ProtoMajor == 2 && strings.HasPrefix(
				r.Header.Get("Content-Type"), "application/grpc") {
				grpchandler.ServeHTTP(w, r)
			} else {
				httphandler.ServeHTTP(w, r)
			}
		})
	}

	s := http.Server{Handler: httphandler, Addr: listen, TLSConfig: tls}
	go func() {
		<-ctx.Done()
		log.Info("shutting down server")
		s.Close()
	}()
	if s.TLSConfig != nil {
		// http2 support with tls enabled
		http2.ConfigureServer(&s, &http2.Server{})
		log.Info("listen on https", "addr", listen)
		return s.ListenAndServeTLS("", "")
	} else {
		// http2 support without https
		s.Handler = h2c.NewHandler(s.Handler, &http2.Server{})
		log.Info("listen on http", "addr", listen)
		return s.ListenAndServe()
	}
}

func ListenAndServeContext(ctx context.Context, listen string, tls *tls.Config, handler http.Handler) error {
	return ListenAndServeContextGRPCAndHTTP(ctx, listen, tls, handler, nil)
}
