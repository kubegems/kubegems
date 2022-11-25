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
	"errors"

	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/edge/common"
	"kubegems.io/kubegems/pkg/edge/options"
	"kubegems.io/kubegems/pkg/edge/tunnel"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/pprof"
)

func Run(ctx context.Context, opts *options.AgentOptions) error {
	return run(ctx, opts)
}

func run(ctx context.Context, options *options.AgentOptions) error {
	ctx = log.NewContext(ctx, log.LogrLogger)
	if options.ClientID == "" {
		return errors.New("--clientid is required")
	}
	tlsConfig, err := options.TLS.ToTLSConfig()
	if err != nil {
		return err
	}
	rest, err := kube.AutoClientConfig()
	if err != nil {
		return err
	}
	c, err := cluster.NewClusterAndStart(ctx, rest)
	if err != nil {
		return err
	}
	sv, err := c.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	upstreamAnnotations := map[string]string{
		common.AnnotationKeyEdgeAgentAddress:         "http://127.0.0.1" + options.Listen,
		common.AnnotationKeyEdgeAgentRegisterAddress: options.EdgeHubAddr,
		common.AnnotationKeyAPIserverAddress:         rest.Host,
		common.AnnotationKeyKubernetesVersion:        sv.String(),
	}

	grpctunnel := tunnel.GrpcTunnelServer{
		TunnelServer: tunnel.NewTunnelServer(options.ClientID, nil),
	}
	httpapi := &AgentAPI{cluster: c}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return grpctunnel.ConnectUpstreamWithRetry(ctx, options.EdgeHubAddr, tlsConfig, "", upstreamAnnotations)
	})
	eg.Go(func() error {
		return httpapi.Run(ctx, options.Listen)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	return eg.Wait()
}
