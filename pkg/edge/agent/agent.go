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
	"fmt"
	"net/http"
	"net/http/httputil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"kubegems.io/kubegems/pkg/edge/options"
	"kubegems.io/kubegems/pkg/log"
	appoptions "sigs.k8s.io/apiserver-network-proxy/cmd/agent/app/options"
	"sigs.k8s.io/apiserver-network-proxy/konnectivity-client/pkg/client"
	"sigs.k8s.io/apiserver-network-proxy/pkg/agent"
)

func Run(ctx context.Context, opts *options.AgentOptions) error {
	return runTranport(ctx, opts)

	tlsConfig, err := opts.TLS.ToTLSConfig()
	if err != nil {
		return err
	}

	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			PermitWithoutStream: true,
		}),
	}
	o := appoptions.NewGrpcProxyAgentOptions()
	cc := &agent.ClientSetConfig{
		Address:                 opts.EdgeHubAddr,
		AgentID:                 o.AgentID,
		AgentIdentifiers:        o.AgentIdentifiers,
		SyncInterval:            o.SyncInterval,
		ProbeInterval:           o.ProbeInterval,
		SyncIntervalCap:         o.SyncIntervalCap,
		DialOptions:             dialOptions,
		ServiceAccountTokenPath: o.ServiceAccountTokenPath,
		WarnOnChannelLimit:      o.WarnOnChannelLimit,
		SyncForever:             o.SyncForever,
	}

	cs := cc.NewAgentClientSet(ctx.Done())
	cs.Serve()

	<-ctx.Done()
	return nil
}

func runTranport(ctx context.Context, opts *options.AgentOptions) error {
	tlsConfig, err := opts.TLS.ToTLSConfig()
	if err != nil {
		return err
	}

	transportCreds := credentials.NewTLS(tlsConfig)
	dialOption := grpc.WithTransportCredentials(transportCreds)
	serverAddress := opts.EdgeHubAddr
	tunnel, err := client.CreateSingleUseGrpcTunnelWithContext(ctx, ctx, serverAddress, dialOption)
	if err != nil {
		return fmt.Errorf("failed to create tunnel %s, got %v", serverAddress, err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: tunnel.DialContext,
		},
	}

	resp, err := client.Get("https://10.21.32.21:443")
	if err != nil {
		log.Error(err, "do http")
		return err
	}
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return err
	}
	fmt.Print(string(dump))

	return nil
}
