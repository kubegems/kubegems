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

	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/edge/common"
	"kubegems.io/kubegems/pkg/edge/options"
	"kubegems.io/kubegems/pkg/edge/tunnel"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/httputil/apiutil"
	"kubegems.io/kubegems/pkg/utils/pprof"
	"kubegems.io/kubegems/pkg/utils/system"
)

func Run(ctx context.Context, options *options.ServerOptions) error {
	server, err := NewEdgeServer(options)
	if err != nil {
		return err
	}
	return server.Run(ctx)
}

type EdgeServer struct {
	server    *tunnel.GrpcTunnelServer
	clusters  *common.EdgeClusterManager
	tlsConfig *tls.Config
	options   *options.ServerOptions
}

func NewEdgeServer(options *options.ServerOptions) (*EdgeServer, error) {
	tlsConfig, err := options.TLS.ToTLSConfig()
	if err != nil {
		return nil, err
	}
	store, err := common.NewLocalK8sStore("")
	if err != nil {
		return nil, err
	}
	server := &EdgeServer{
		server: &tunnel.GrpcTunnelServer{
			TunnelServer: tunnel.NewTunnelServer(options.ServerID, nil),
		},
		tlsConfig: tlsConfig,
		options:   options,
		clusters:  common.NewClusterManager(store, options.Host),
	}
	return server, nil
}

func (s *EdgeServer) Run(ctx context.Context) error {
	ctx = log.NewContext(ctx, log.LogrLogger)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.server.Serve(ctx, s.options.ListenGrpc, s.tlsConfig)
	})
	eg.Go(func() error {
		return s.RunHTTPAPI(ctx, s.options.Listen, nil)
	})
	eg.Go(func() error {
		return s.clusters.SyncTunnelStatusFrom(ctx, s.server.TunnelServer)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	return eg.Wait()
}

func (s *EdgeServer) RunHTTPAPI(ctx context.Context, listen string, tls *tls.Config) error {
	edgehubapi := &common.EdgeClusterAPI{
		Cluster: s.clusters,
		Tunnel:  s.server.TunnelServer,
	}
	return system.ListenAndServeContext(ctx, listen, tls, apiutil.NewRestfulAPI("v1", nil, []apiutil.RestModule{
		edgehubapi,
	}))
}
