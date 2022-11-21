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

package agent

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"os"

	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/edge/common"
	"kubegems.io/kubegems/pkg/edge/options"
	"kubegems.io/kubegems/pkg/edge/tunnel"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/pprof"
)

func Run(ctx context.Context, opts *options.AgentOptions) error {
	s, err := New(opts)
	if err != nil {
		return err
	}
	return s.Run(ctx)
}

type EdgeAgent struct {
	tunnel.GrpcTunnelServer
	upstreamAnnotations tunnel.Annotations
	tlsConfig           *tls.Config
	options             *options.AgentOptions
	api                 *AgentAPI
}

func New(options *options.AgentOptions) (*EdgeAgent, error) {
	if options.ClientID == "" {
		return nil, errors.New("--clientid is required")
	}
	tlsConfig, err := options.TLS.ToTLSConfig()
	if err != nil {
		return nil, err
	}
	annotations := map[string]string{
		common.AnnotationKeyEdgeAgentAddress: "http://127.0.0.1" + options.Listen,
		common.AnnotationKeyAPIserverAddress: "https://" + net.JoinHostPort(
			os.Getenv("KUBERNETES_SERVICE_HOST"),
			os.Getenv("KUBERNETES_SERVICE_PORT"),
		),
	}
	agent := &EdgeAgent{
		tlsConfig:           tlsConfig,
		upstreamAnnotations: annotations,
		options:             options,
		api:                 &AgentAPI{},
		GrpcTunnelServer: tunnel.GrpcTunnelServer{
			TunnelServer: tunnel.NewTunnelServer(options.ClientID, nil),
		},
	}
	return agent, nil
}

func (s *EdgeAgent) Run(ctx context.Context) error {
	ctx = log.NewContext(ctx, log.LogrLogger)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.GrpcTunnelServer.ConnectUpstream(ctx, s.options.EdgeHubAddr, s.tlsConfig, "", s.upstreamAnnotations)
	})
	eg.Go(func() error {
		return s.api.Run(ctx, s.options.Listen)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	return eg.Wait()
}
