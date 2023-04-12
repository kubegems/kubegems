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

package server

import (
	"context"
	"crypto/tls"
	"net/http"

	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/edge/tunnel"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/httputil/apiutil"
	"kubegems.io/kubegems/pkg/utils/pprof"
	"kubegems.io/kubegems/pkg/utils/system"
)

func Run(ctx context.Context, options *Options) error {
	server, err := NewEdgeServer(ctx, options)
	if err != nil {
		return err
	}
	return server.Run(ctx)
}

type EdgeServer struct {
	server    *tunnel.GrpcTunnelServer
	clusters  *EdgeManager
	tlsConfig *tls.Config
	options   *Options
}

func NewEdgeServer(ctx context.Context, options *Options) (*EdgeServer, error) {
	tlsConfig, err := options.TLS.ToTLSConfig()
	if err != nil {
		return nil, err
	}
	edgemanager, err := NewClusterManager(ctx, "", options.Host)
	if err != nil {
		return nil, err
	}
	server := &EdgeServer{
		server: &tunnel.GrpcTunnelServer{
			TunnelServer: tunnel.NewTunnelServer(options.ServerID, nil),
		},
		tlsConfig: tlsConfig,
		options:   options,
		clusters:  edgemanager,
	}
	return server, nil
}

func (s *EdgeServer) Run(ctx context.Context) error {
	ctx = log.NewContext(ctx, log.LogrLogger)
	eg, ctx := errgroup.WithContext(ctx)

	if s.options.Listen == s.options.ListenGrpc {
		eg.Go(func() error {
			return system.ListenAndServeContextGRPCAndHTTP(ctx,
				s.options.Listen,
				s.tlsConfig,
				s.HTTPAPI(),
				s.server.GrpcServer(s.tlsConfig),
			)
		})
	} else {
		eg.Go(func() error {
			return s.server.ServeGrpc(ctx, s.options.ListenGrpc, s.tlsConfig)
		})
		eg.Go(func() error {
			return system.ListenAndServeContext(ctx, s.options.Listen, nil, s.HTTPAPI())
		})
	}
	eg.Go(func() error {
		return s.clusters.SyncTunnelStatusFrom(ctx, s.server.TunnelServer)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	return eg.Wait()
}

func (s *EdgeServer) HTTPAPI() http.Handler {
	edgeapi := &EdgeClusterAPI{
		Cluster: s.clusters,
		Tunnel:  s.server.TunnelServer,
	}
	return apiutil.NewRestfulAPI("v1", nil, []apiutil.RestModule{edgeapi})
}
