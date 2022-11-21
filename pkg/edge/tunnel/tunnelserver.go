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
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"kubegems.io/kubegems/pkg/log"
)

var DefaultDialTimeout = 30 * time.Second

type TunnelServer struct {
	auth        AuthenticationManager
	id          string
	connections *ConnectionManager
	routeTable  *RouteTable
	eventer     *TunnelEventer
}

func NewTunnelServer(id string, auth AuthenticationManager) *TunnelServer {
	if auth == nil {
		auth = &NonAuthManager{}
	}
	s := &TunnelServer{
		id:   id,
		auth: auth,
	}
	s.routeTable = NewEmptyRouteTable(s)
	s.connections = NewConectionManager(s)
	s.eventer = NewTunnelEventer(s)
	return s
}

func (s *TunnelServer) Connect(ctx context.Context, channel Tunnel, token string, annotations Annotations, options TunnelOptions) error {
	connectedChannel, err := s.authStage(ctx, channel, token)
	if err != nil {
		return err
	}
	connectedChannel.Options = options
	routedata, err := s.routeExchangeStage(ctx, connectedChannel, annotations)
	if err != nil {
		return err
	}
	// connected
	s.routeTable.Connect(connectedChannel, *routedata)
	defer s.routeTable.Disconnect(connectedChannel)

	for {
		pkt := new(Packet)
		if err := connectedChannel.Recv(pkt); err != nil {
			return err
		}
		s.preRouting(connectedChannel, pkt)
	}
}

func (s *TunnelServer) authStage(ctx context.Context, channel Tunnel, token string) (*ConnectedTunnel, error) {
	// send meta and auth
	connectData := PacketDataConnect{Token: token}
	log.Info("connect send", "data", connectData)
	if err := channel.Send(&Packet{
		Kind: PacketKindConnect,
		Src:  s.id,
		Data: PacketEncode(connectData),
	}); err != nil {
		return nil, err
	}
	// wait remote meta and auth
	connectpkt := new(Packet)
	if err := channel.Recv(connectpkt); err != nil {
		return nil, err
	}
	if connectpkt.Kind != PacketKindConnect {
		return nil, fmt.Errorf("invalid packet kind: %v", connectpkt.Kind)
	}

	remoteid := connectpkt.Src
	connectData = PacketDecode[PacketDataConnect](connectpkt.Data)
	log.Info("connect recv", "remote", remoteid, "data", connectData)
	// check not empty remote id
	if remoteid == "" {
		err := errors.New("empty tunnel id")
		_ = channel.Send(&Packet{Kind: PacketKindClose, Error: err.Error()})
		return nil, err
	}
	// check remote auth
	if err := s.auth.Authentication(ctx, remoteid, connectData.Token); err != nil {
		_ = channel.Send(&Packet{Kind: PacketKindClose, Error: err.Error()})
		log.Error(err, "auth faild", "remote", remoteid, "token", connectData.Token)
		return nil, err
	}
	// send ack
	if err := channel.Send(&Packet{Kind: PacketKindData, Src: s.id, Dest: remoteid}); err != nil {
		return nil, err
	}
	// wait ack
	ackpkt := new(Packet)
	if err := channel.Recv(ackpkt); err != nil {
		return nil, err
	}
	if ackpkt.Kind == PacketKindClose || ackpkt.Error != "" {
		return nil, fmt.Errorf("remote channel closed: %s", ackpkt.Error)
	}
	log.Info("auth success", "remote", remoteid)
	// connected
	return &ConnectedTunnel{Tunnel: channel, ID: remoteid}, nil
}

func (s *TunnelServer) routeExchangeStage(ctx context.Context, idchannel *ConnectedTunnel, annotationsToSend Annotations) (*PacketDataRoute, error) {
	data := PacketDataRoute{
		Kind:        RouteUpdateKindInit,
		Annotations: annotationsToSend,
	}
	if idchannel.Options.SendRouteChange {
		data.Peers = s.routeTable.allRechablePeers()
	}
	log.Info("route send", "dest", idchannel.ID, "data", data)
	// advetise self peers
	if err := idchannel.Send(&Packet{
		Kind: PacketKindRoute,
		Src:  s.id,
		Dest: idchannel.ID,
		Data: PacketEncode(data),
	}); err != nil {
		return nil, err
	}
	// wait remote route
	routepkt := &Packet{}
	if err := idchannel.Recv(routepkt); err != nil {
		return nil, err
	}
	if routepkt.Kind != PacketKindRoute {
		return nil, fmt.Errorf("unexpect packet type %d", routepkt.Kind)
	}
	routedata := PacketDecode[PacketDataRoute](routepkt.Data)
	log.Info("route recv", "src", routepkt.Src, "data", routedata)
	return &routedata, nil
}

func (s *TunnelServer) preRouting(income *ConnectedTunnel, pkt *Packet) {
	if pkt.Dest == "" {
		// empty dest is to local
		pkt.Dest = s.id
	}
	if pkt.Dest != s.id {
		s.forward(income, pkt)
	} else {
		s.localIn(income, pkt)
	}
}

func (s *TunnelServer) forward(income *ConnectedTunnel, pkt *Packet) error {
	log.Info("packet forward", "src", pkt.Src, "dest", pkt.Dest)
	targetPeer, err := s.routeTable.Select(pkt.Dest)
	if err != nil {
		_ = income.Send(&Packet{
			Kind:    PacketKindClose,
			Src:     s.id,
			Dest:    pkt.Src,
			DestCID: pkt.SrcCID,
			Error:   err.Error(),
		})
		log.Error(err, "choose")
		return err
	}
	if err := targetPeer.Send(pkt); err != nil {
		_ = income.Send(&Packet{
			Kind:    PacketKindClose,
			Src:     s.id,
			Dest:    pkt.Src,
			DestCID: pkt.SrcCID,
			Error:   err.Error(),
		})
		log.Error(err, "forward")
		return err
	}
	return nil
}

func (s *TunnelServer) localIn(channel *ConnectedTunnel, pkt *Packet) {
	log.V(5).Info("packet in", "src", pkt.Src, "dest", pkt.Dest)
	switch pkt.Kind {
	case PacketKindOpen:
		go s.connections.accept(channel, pkt.Src, pkt.SrcCID, PacketDecode[PacketDataOpen](pkt.Data))
	case PacketKindData:
		go func() {
			if err := s.connections.ack(channel, pkt.Src, pkt.SrcCID, pkt.DestCID, pkt.Data, pkt.Error); err != nil {
				_ = channel.Send(&Packet{
					Kind:    PacketKindClose,
					SrcCID:  pkt.DestCID,
					Src:     s.id,
					Dest:    pkt.Src,
					DestCID: pkt.SrcCID,
					Error:   err.Error(),
				})
			}
		}()
	case PacketKindClose:
		go s.connections.close(channel, pkt.Src, pkt.SrcCID, pkt.DestCID)
	case PacketKindRoute:
		go s.routeTable.Update(pkt.Src, PacketDecode[PacketDataRoute](pkt.Data))
	}
}

type Dailer struct {
	remote string
	server *TunnelServer
}

func (s *TunnelServer) DialerOn(dest string) Dailer {
	return Dailer{server: s, remote: dest}
}

func (d Dailer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return d.server.connections.Open(network, address, timeout, d.remote)
}

func (d Dailer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if deadline, ok := ctx.Deadline(); ok {
		return d.DialTimeout(network, address, time.Until(deadline))
	} else {
		return d.DialTimeout(network, address, DefaultDialTimeout)
	}
}

func (s *TunnelServer) Wacth(ctx context.Context) EventWatcher {
	return s.eventer.Watch(ctx)
}

func RandomServerID(prefix string) string {
	return prefix + strings.ReplaceAll(uuid.NewString(), "-", "")
}
