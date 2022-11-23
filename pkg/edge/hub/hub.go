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
	"time"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/wait"
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
	store, err := common.NewLocalK8sStore(options.CurrentNamespace)
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
		clusters:  common.NewClusterManager(store, ""),
		tlsConfig: tlsConfig,
		options:   options,
	}
	return hub, nil
}

type EdgeHubServer struct {
	tunnel.GrpcTunnelServer
	clusters            *common.EdgeClusterManager
	tlsConfig           *tls.Config
	options             *options.HubOptions
	upstreamAnnotations tunnel.Annotations
}

func (s *EdgeHubServer) Run(ctx context.Context) error {
	ctx = log.NewContext(ctx, log.LogrLogger)

	eg, ctx := errgroup.WithContext(ctx)
	if s.options.Listen == s.options.ListenGrpc {
		eg.Go(func() error {
			return s.RunHTTPGRPCMux(ctx)
		})
	} else {
		eg.Go(func() error {
			return s.RunHTTPAPI(ctx, s.options.Listen)
		})
		eg.Go(func() error {
			return s.ServeGrpc(ctx, s.options.ListenGrpc, s.tlsConfig)
		})
	}
	if s.options.CurrentNamespace != "" {
		eg.Go(func() error {
			return s.RunStatusWatcher(ctx)
		})
	}
	eg.Go(func() error {
		c := s.tlsConfig.Clone()
		c.InsecureSkipVerify = true
		return s.ConnectUpstream(ctx, s.options.EdgeServerAddr, c, "", s.upstreamAnnotations)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	return eg.Wait()
}

func (s *EdgeHubServer) RunHTTPGRPCMux(ctx context.Context) error {
	return system.ListenAndServeContextGRPCAndHTTP(ctx,
		s.options.ListenGrpc,
		s.tlsConfig,
		s.HTTPHandler(),
		s.GrpcServer(s.tlsConfig))
}

func (s *EdgeHubServer) HTTPHandler() http.Handler {
	edgehubapi := &common.EdgeClusterAPI{
		Cluster: s.clusters,
		Tunnel:  s.TunnelServer,
	}
	return apiutil.NewRestfulAPI("v1", nil, []apiutil.RestModule{
		edgehubapi,
	})
}

func (s *EdgeHubServer) RunHTTPAPI(ctx context.Context, listen string) error {
	return system.ListenAndServeContext(ctx, listen, nil, s.HTTPHandler())
}

func (s *EdgeHubServer) RunStatusWatcher(ctx context.Context) error {
	logr.FromContextOrDiscard(ctx).Info("starting status watcher")
	return wait.PollInfiniteWithContext(ctx, time.Second, func(ctx context.Context) (done bool, err error) {
		if err := s.clusters.SyncTunnelStatusFrom(ctx, s.TunnelServer); err != nil {
			log.Error(err, "status watcher")
		}
		return false, nil
	})
}
