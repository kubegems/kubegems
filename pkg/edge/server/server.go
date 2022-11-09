package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"kubegems.io/kubegems/pkg/edge/options"
	"kubegems.io/kubegems/pkg/log"
	"sigs.k8s.io/apiserver-network-proxy/konnectivity-client/proto/client"
	"sigs.k8s.io/apiserver-network-proxy/pkg/server"
)

func Run(ctx context.Context, options *options.ServerOptions) error {
	// proxy server
	server := server.NewProxyServer(
		defaultServerID(),
		[]server.ProxyStrategy{
			server.ProxyStrategyDefault,
			server.ProxyStrategyDefaultRoute,
			server.ProxyStrategyDestHost,
		},
		1,
		&server.AgentTokenAuthenticationOptions{Enabled: false})

	// grpc
	tlsConfig, err := options.TLS.ToTLSConfig()
	if err != nil {
		return err
	}
	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: time.Hour}),
	)

	client.RegisterProxyServiceServer(grpcServer, server)

	lis, err := net.Listen("tcp", options.Listen)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", options.Listen, err)
	}
	log.Info("listen on", "address", options.Listen)
	go grpcServer.Serve(lis)
	<-ctx.Done()
	grpcServer.GracefulStop()
	return nil
}

func defaultServerID() string {
	if id := os.Getenv("PROXY_SERVER_ID"); id != "" {
		return id
	}
	return uuid.New().String()
}
