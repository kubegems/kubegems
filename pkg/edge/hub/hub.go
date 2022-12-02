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

package hub

import (
	"context"
	"crypto/tls"
	"net/http"

	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/edge/common"
	"kubegems.io/kubegems/pkg/edge/options"
	"kubegems.io/kubegems/pkg/edge/tunnel"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/config"
	"kubegems.io/kubegems/pkg/utils/httputil/apiutil"
	"kubegems.io/kubegems/pkg/utils/pprof"
	"kubegems.io/kubegems/pkg/utils/system"
)

func Run(ctx context.Context, options *options.HubOptions) error {
	server, err := New(options)
	if err != nil {
		return err
	}
	return server.Run(ctx)
}

func New(options *options.HubOptions) (*EdgeHubServer, error) {
	if err := config.Validate(options); err != nil {
		return nil, err
	}
	tlsConfig, err := options.TLS.ToTLSConfig()
	if err != nil {
		return nil, err
	}
	cert, key := common.EncodeToX509Pair(tlsConfig.Certificates[0])
	hub := &EdgeHubServer{
		upstreamAnnotations: map[string]string{
			common.AnnotationKeyEdgeHubAddress: options.Host,
			common.AnnotationKeyEdgeHubCert:    string(cert),
			common.AnnotationKeyEdgeHubKey:     string(key),
		},
		GrpcTunnelServer: tunnel.GrpcTunnelServer{
			TunnelServer: tunnel.NewTunnelServer(options.ServerID, nil),
		},
		tlsConfig: tlsConfig,
		options:   options,
	}
	return hub, nil
}

type EdgeHubServer struct {
	tunnel.GrpcTunnelServer
	tlsConfig           *tls.Config
	options             *options.HubOptions
	upstreamAnnotations tunnel.Annotations
}

func (s *EdgeHubServer) Run(ctx context.Context) error {
	ctx = log.NewContext(ctx, log.LogrLogger)

	eg, ctx := errgroup.WithContext(ctx)

	if s.options.Listen == s.options.ListenGrpc {
		eg.Go(func() error {
			return system.ListenAndServeContextGRPCAndHTTP(
				ctx, s.options.Listen, s.tlsConfig,
				s.HTTPAPI(),
				s.GrpcServer(s.tlsConfig),
			)
		})
	} else {
		eg.Go(func() error {
			return s.ServeGrpc(ctx, s.options.ListenGrpc, s.tlsConfig)
		})
		eg.Go(func() error {
			return system.ListenAndServeContext(ctx, s.options.Listen, nil, s.HTTPAPI())
		})
	}
	eg.Go(func() error {
		c := s.tlsConfig.Clone()
		c.InsecureSkipVerify = true
		return s.ConnectUpstreamWithRetry(ctx, s.options.EdgeServerAddr, c, "", s.upstreamAnnotations)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	return eg.Wait()
}

func (s *EdgeHubServer) HTTPAPI() http.Handler {
	// handler provides a health check endpoint
	return apiutil.NewRestfulAPI("", nil, nil)
}
