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

package pprof

import (
	"context"
	"expvar"
	"net"
	"net/http"
	"net/http/pprof"
	"os"

	"kubegems.io/kubegems/pkg/log"
)

// ServeDebug provides a debug endpoint
func newHandler() http.Handler {
	// don't use the default http server mux to make sure nothing gets registered
	// that we don't want to expose via containerd
	m := http.NewServeMux()
	m.Handle("/debug/vars", expvar.Handler())
	m.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	m.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	m.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	m.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	m.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	return m
}

func Run(ctx context.Context) error {
	var port string
	port = os.Getenv("GEMS_PPROF_PORT")
	if port == "" {
		port = ":6060"
	}
	server := http.Server{
		Addr:    port,
		Handler: newHandler(),
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
	log := log.FromContextOrDiscard(ctx)

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(ctx)
		log.Info("pprof stopped")
	}()
	log.Info("debug pprof listen", "addr", server.Addr)
	return server.ListenAndServe()
}
