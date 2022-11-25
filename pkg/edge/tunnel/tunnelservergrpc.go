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

package tunnel

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"k8s.io/apimachinery/pkg/util/wait"
	"kubegems.io/kubegems/pkg/edge/tunnel/proto"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/system"
)

const (
	DefaultRetryInterval = 10 * time.Second
)

type Options struct {
	PeerID          string `json:"peerID,omitempty"`
	Listen          string `json:"listen,omitempty"`
	UpstreamAddr    string `json:"upstreamAddr,omitempty"`
	EnableClientTLS bool   `json:"enableClientTLS,omitempty"`
	Token           string `json:"token,omitempty"`
	TLS             *TLS   `json:"tls,omitempty"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Listen: "",
		PeerID: uuid.NewString(),
		TLS:    NewDefaultTLS(),
	}
}

type grpcstream interface {
	Send(*proto.Packet) error
	Recv() (*proto.Packet, error)
}

type GRPCTunnel[T grpcstream] struct {
	inner T
}

func (t *GRPCTunnel[T]) Recv(into *Packet) error {
	pkt, err := t.inner.Recv()
	if err != nil {
		return err
	}
	into.Kind = PacketKind(pkt.Kind)
	into.Data = pkt.Data
	into.Error = pkt.Error
	into.Dest = pkt.Dest
	into.DestCID = pkt.DestID
	into.Src = pkt.Src
	into.SrcCID = pkt.SrcID
	return nil
}

func (t *GRPCTunnel[T]) Send(from *Packet) error {
	return t.inner.Send(&proto.Packet{
		Src:    from.Src,
		SrcID:  from.SrcCID,
		Dest:   from.Dest,
		DestID: from.DestCID,
		Kind:   int64(from.Kind),
		Data:   from.Data,
		Error:  from.Error,
	})
}

func Run(ctx context.Context, options *Options) error {
	ctx = logr.NewContext(ctx, log.LogrLogger)
	log.Info("init", "options", options)

	tlsConfig, err := options.TLS.ToTLSConfig()
	if err != nil {
		return err
	}
	server := GrpcTunnelServer{
		TunnelServer: NewTunnelServer(options.PeerID, nil),
	}
	eg := errgroup.Group{}
	if listen := options.Listen; listen != "" {
		eg.Go(func() error {
			return server.ServeGrpc(ctx, options.Listen, tlsConfig)
		})
	}
	if updtream := options.UpstreamAddr; updtream != "" {
		eg.Go(func() error {
			return server.ConnectUpstreamWithRetry(ctx, options.UpstreamAddr, tlsConfig, options.Token, nil)
		})
	}
	return eg.Wait()
}

type GrpcTunnelServer struct {
	TunnelServer      *TunnelServer
	ClientAnnotations Annotations // annotations send to downstream clients
	proto.UnimplementedPeerServiceServer
}

func (s GrpcTunnelServer) Connect(connectServer proto.PeerService_ConnectServer) error {
	return s.TunnelServer.Connect(
		connectServer.Context(), // context
		&GRPCTunnel[proto.PeerService_ConnectServer]{inner: connectServer}, // grpc based tunnel
		"",                                    // server do't provide a token to client, required no auth for client.
		s.ClientAnnotations,                   // annotations send to downstream clients
		TunnelOptions{SendRouteChange: false}) // we don't send route update to downstream channels.
}

func (s GrpcTunnelServer) GrpcServer(tlsConfig *tls.Config) *grpc.Server {
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: time.Hour}),
		grpc.Creds(credentials.NewTLS(tlsConfig)),
	)
	proto.RegisterPeerServiceServer(grpcServer, s)
	return grpcServer
}

func (s GrpcTunnelServer) ServeGrpc(ctx context.Context, listen string, tlsConfig *tls.Config) error {
	return system.ListenAndServeGRPCContext(ctx, listen, s.GrpcServer(tlsConfig))
}

func (s GrpcTunnelServer) ConnectUpstreamWithRetry(ctx context.Context, addr string, tlsConfig *tls.Config, token string, annotations Annotations) error {
	return wait.PollImmediateInfiniteWithContext(ctx, DefaultRetryInterval, func(ctx context.Context) (done bool, err error) {
		if err := s.ConnectUpstream(ctx, addr, tlsConfig, token, annotations); err != nil {
			log.Error(err, "on connect upstream")
		}
		return false, nil
	})
}

func (s GrpcTunnelServer) ConnectUpstream(ctx context.Context, addr string, tlsConfig *tls.Config, token string, annotations Annotations) error {
	log.FromContextOrDiscard(ctx).Info("connecting upstream", "addr", addr, "annotations", annotations)
	dialoptions := []grpc.DialOption{}
	if tlsConfig != nil {
		dialoptions = append(dialoptions,
			grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		)
	} else {
		dialoptions = append(dialoptions,
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
			})),
		)
	}
	c, err := grpc.DialContext(ctx, addr, dialoptions...)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		c.Close()
	}()

	stream, err := proto.NewPeerServiceClient(c).Connect(ctx)
	if err != nil {
		return err
	}
	peer := &GRPCTunnel[proto.PeerService_ConnectClient]{inner: stream}
	return s.TunnelServer.Connect(ctx, peer, token, annotations, TunnelOptions{
		SendRouteChange: true,
		IsDefaultOut:    true, // as default out if no route info
	})
}
