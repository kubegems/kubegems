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
