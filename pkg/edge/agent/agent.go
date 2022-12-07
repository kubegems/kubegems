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
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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

type EdgeAgent struct {
	config      *rest.Config
	cluster     cluster.Interface
	tunserver   tunnel.GrpcTunnelServer
	httpapi     *AgentAPI
	options     *options.AgentOptions
	annotations tunnel.Annotations
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
	ea := &EdgeAgent{
		config:      rest,
		options:     options,
		annotations: nil,
		cluster:     c,
		httpapi:     &AgentAPI{cluster: c},
		tunserver:   tunnel.GrpcTunnelServer{TunnelServer: tunnel.NewTunnelServer(options.ClientID, nil)},
	}
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return ea.tunserver.ConnectUpstreamWithRetry(ctx, options.EdgeHubAddr, tlsConfig, "", ea.getAnnotations(ctx))
	})
	eg.Go(func() error {
		return ea.RunKeepAliveRouter(ctx, ea.options.KeepAliveInterval, ea.getAnnotations)
	})
	eg.Go(func() error {
		return ea.httpapi.Run(ctx, options.Listen)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	return eg.Wait()
}

func (ea *EdgeAgent) RunKeepAliveRouter(ctx context.Context, duration time.Duration, annotationsfunc func(ctx context.Context) tunnel.Annotations) error {
	log.Info("starting refresh router")

	if duration <= 0 {
		duration = 30 * time.Second
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			timer.Reset(duration)
			annotations := annotationsfunc(ctx)
			ea.tunserver.TunnelServer.SendKeepAlive(ctx, annotations)
		}
	}
}

func (ea *EdgeAgent) getAnnotations(ctx context.Context) tunnel.Annotations {
	if ea.annotations != nil {
		return ea.annotations
	}
	sv, _ := ea.cluster.Discovery().ServerVersion()
	nodeList, _ := ea.cluster.Kubernetes().CoreV1().Nodes().List(ctx, v1.ListOptions{})
	annotations := map[string]string{
		common.AnnotationKeyEdgeAgentAddress:           "http://127.0.0.1" + ea.options.Listen,
		common.AnnotationKeyEdgeAgentRegisterAddress:   ea.options.EdgeHubAddr,
		common.AnnotationKeyEdgeAgentKeepaliveInterval: ea.options.KeepAliveInterval.String(),
		common.AnnotationKeyAPIserverAddress:           ea.config.Host,
		common.AnnotationKeyKubernetesVersion:          sv.String(),
		common.AnnotationKeyNodesCount:                 strconv.Itoa(len(nodeList.Items)),
	}
	ea.annotations = annotations
	return annotations
}
