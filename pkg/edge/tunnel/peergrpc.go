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
	"fmt"
	"net"
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
)

type Options struct {
	PeerID        string `json:"peerID,omitempty"`
	Listen        string `json:"listen,omitempty"`
	UpstreamAddr  string `json:"upstreamAddr,omitempty"`
	UpstreamToken string `json:"upstreamToken,omitempty"`
	TLS           *TLS   `json:"tls,omitempty"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Listen: "",
		PeerID: uuid.NewString(),
		TLS:    NewDefaultTLS(),
	}
}

type stream interface {
	Send(*proto.Packet) error
	Recv() (*proto.Packet, error)
	Context() context.Context
}

type GRPCPeer[T stream] struct {
	id    string
	inner T
}

func (t *GRPCPeer[T]) ID() string {
	return t.id
}

func (t *GRPCPeer[T]) Recv(into *Packet) error {
	pkt, err := t.inner.Recv()
	if err != nil {
		return err
	}
	into.Kind = PacketKind(pkt.Kind)
	into.Data = pkt.Data
	into.Error = pkt.Error
	into.Dest = pkt.Dest
	into.DestID = pkt.DestID
	into.Src = pkt.Src
	into.SrcID = pkt.SrcID
	return nil
}

func (t *GRPCPeer[T]) Send(from *Packet) error {
	return t.inner.Send(&proto.Packet{
		Src:    from.Src,
		SrcID:  from.SrcID,
		Dest:   from.Dest,
		DestID: from.DestID,
		Kind:   int64(from.Kind),
		Data:   from.Data,
		Error:  from.Error,
	})
}

func (t *GRPCPeer[T]) Done() <-chan struct{} {
	return t.inner.Context().Done()
}

func (t *GRPCPeer[T]) Close() {
}

func Run(ctx context.Context, options *Options) error {
	ctx = logr.NewContext(ctx, log.LogrLogger)
	log.Info("init", "options", options)
	server := GrpcPeerServer{
		PeerServer: NewPeerServer(
			options.PeerID,
			options.UpstreamToken,
			PeerOptions{
				ClientOnly: options.Listen == "",
			},
		),
	}

	eg := errgroup.Group{}
	if listen := options.Listen; listen != "" {
		eg.Go(func() error {
			return server.Serve(ctx, options)
		})
	}
	if updtream := options.UpstreamAddr; updtream != "" {
		eg.Go(func() error {
			return server.ConnectUpstream(ctx, options)
		})
	}
	return eg.Wait()
}

type GrpcPeerServer struct {
	PeerServer *PeerServer
	proto.UnimplementedPeerServiceServer
}

func (s GrpcPeerServer) Connect(connectServer proto.PeerService_ConnectServer) error {
	peer := &GRPCPeer[proto.PeerService_ConnectServer]{inner: connectServer}
	return s.PeerServer.PeerConnect(connectServer.Context(), peer)
}

func (s GrpcPeerServer) Serve(ctx context.Context, options *Options) error {
	tlsConfig, err := options.TLS.ToTLSConfig()
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: time.Hour}),
	)
	proto.RegisterPeerServiceServer(grpcServer, s)
	lis, err := net.Listen("tcp", options.Listen)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", options.Listen, err)
	}
	go func() {
		<-ctx.Done()
		grpcServer.Stop()
	}()
	log.Info("grpc server started", "listen", lis.Addr().String())
	return grpcServer.Serve(lis)
}

func (s GrpcPeerServer) ConnectUpstream(ctx context.Context, options *Options) error {
	tlsConfig, err := options.TLS.ToTLSConfig()
	if err != nil {
		return err
	}
	addr := options.UpstreamAddr
	return wait.PollInfiniteWithContext(ctx, time.Second, func(ctx context.Context) (done bool, err error) {
		if err := s.connectUpstream(ctx, addr, tlsConfig, options.UpstreamToken); err != nil {
			log.Error(err, "connect upstream")
		}
		return false, nil
	})
}

func (s GrpcPeerServer) connectUpstream(ctx context.Context, addr string, tlsConfig *tls.Config, token string) error {
	c, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(
			credentials.NewTLS(tlsConfig),
		),
	)
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
	peer := &GRPCPeer[proto.PeerService_ConnectClient]{inner: stream}
	return s.PeerServer.PeerConnect(ctx, peer)
}
