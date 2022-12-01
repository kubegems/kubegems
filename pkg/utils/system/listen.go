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

package system

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
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
		log.Info("shutting down server", "addr", listen)
		s.Close()
	}()
	// nolint: nestif
	if s.TLSConfig != nil {
		// http2 support with tls enabled
		http2.ConfigureServer(&s, &http2.Server{})
		if grpchandler != nil {
			log.Info("starting https(grpc) server", "addr", listen)
		} else {
			log.Info("starting https server", "addr", listen)
		}
		return s.ListenAndServeTLS("", "")
	} else {
		// http2 support without https
		s.Handler = h2c.NewHandler(s.Handler, &http2.Server{})
		if grpchandler != nil {
			log.Info("starting http(grpc) server", "addr", listen)
		} else {
			log.Info("starting http server", "addr", listen)
		}
		return s.ListenAndServe()
	}
}

func ListenAndServeContext(ctx context.Context, listen string, tls *tls.Config, handler http.Handler) error {
	return ListenAndServeContextGRPCAndHTTP(ctx, listen, tls, handler, nil)
}

func ListenAndServeGRPCContext(ctx context.Context, listen string, grpcServer *grpc.Server) error {
	log := logr.FromContextOrDiscard(ctx)
	lis, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		grpcServer.Stop()
	}()
	log.Info("starting grpc server", "addr", lis.Addr().String())
	return grpcServer.Serve(lis)
}
