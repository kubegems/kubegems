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
	"time"

	"kubegems.io/kubegems/pkg/log"
)

type Channel interface {
	Recv(*Packet) error
	Send(*Packet) error
	// Done() <-chan struct{}
}

type IDChannel struct {
	ID      string
	Options PeerOptions
	Channel
}

type PeerOptions struct {
	ClientOnly bool `json:"clientOnly,omitempty"` // the peer only receive income connection
}

type PeerServer struct {
	PeerID                string
	Token                 string
	PeerOptions           PeerOptions
	AuthenticationManager AuthenticationManager
	Connections           *ConnectionManager
	RouteTable            *RouteTable
}

func NewPeerServer(peerid string, token string, peerOptions PeerOptions) *PeerServer {
	s := &PeerServer{
		PeerID:                peerid,
		PeerOptions:           peerOptions,
		Token:                 token,
		AuthenticationManager: &NonAuthManager{},
	}
	s.RouteTable = NewEmptyRouteTable(s)
	s.Connections = NewConectionManager(s)
	return s
}

func (s *PeerServer) PeerConnect(ctx context.Context, channel Channel) error {
	idChannel, err := s.authStage(ctx, channel, s.Token)
	if err != nil {
		return err
	}
	routedata, err := s.routeExchangeStage(ctx, idChannel)
	if err != nil {
		return err
	}
	// connected
	s.RouteTable.Connect(idChannel, routedata.SubPeers)
	defer s.RouteTable.Disconnect(idChannel)

	for {
		pkt := new(Packet)
		if err := idChannel.Recv(pkt); err != nil {
			return err
		}
		s.preRouting(idChannel, pkt)
	}
}

func (s *PeerServer) authStage(ctx context.Context, channel Channel, token string) (IDChannel, error) {
	// send meta and auth
	if err := channel.Send(&Packet{
		Kind: PacketKindConnect,
		Src:  s.PeerID,
		Data: PacketEncode(
			PacketDataConnect{
				Token:   token,
				Options: s.PeerOptions,
			}),
	}); err != nil {
		return IDChannel{}, err
	}
	// wait remote meta and auth
	connectpkt := new(Packet)
	if err := channel.Recv(connectpkt); err != nil {
		return IDChannel{}, err
	}
	if connectpkt.Kind != PacketKindConnect {
		return IDChannel{}, fmt.Errorf("invalid packet kind: %v", connectpkt.Kind)
	}

	remoteid := connectpkt.Src
	connectData := PacketDecode[PacketDataConnect](connectpkt.Data)
	log.Info("recv remote connect", "remote", remoteid, "data", connectData)
	// check remote auth
	if err := s.AuthenticationManager.Authentication(ctx, remoteid, connectData.Token); err != nil {
		return IDChannel{}, err
	}
	// send ack
	if err := channel.Send(&Packet{
		Kind: PacketKindData,
		Src:  s.PeerID,
		Dest: remoteid,
	}); err != nil {
		return IDChannel{}, err
	}
	// wait ack
	ackpkt := new(Packet)
	if err := channel.Recv(ackpkt); err != nil {
		return IDChannel{}, err
	}
	log.Info("auth success", "remote", remoteid)
	// connected
	return IDChannel{ID: remoteid, Options: connectData.Options, Channel: channel}, nil
}

func (s *PeerServer) routeExchangeStage(ctx context.Context, idchannel IDChannel) (*PacketDataRoute, error) {
	data := PacketDataRoute{
		Kind: PeerUpdateKindRefresh,
	}
	if !idchannel.Options.ClientOnly {
		data.SubPeers = s.RouteTable.allRechablePeers()
	}
	log.Info("send route", "dest", idchannel.ID, "data", data)
	// advetise self peers
	if err := idchannel.Send(&Packet{
		Kind: PacketKindRoute,
		Src:  s.PeerID,
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
	log.Info("recv route", "src", routepkt.Src, "data", routedata)
	return &routedata, nil
}

func (s *PeerServer) preRouting(income IDChannel, pkt *Packet) {
	if pkt.Dest == "" {
		// empty dest is to local
		pkt.Dest = s.PeerID
	}
	if pkt.Dest != s.PeerID {
		s.forward(income, pkt)
	} else {
		s.localIn(income, pkt)
	}
}

func (s *PeerServer) forward(income IDChannel, pkt *Packet) error {
	log.Info("forward", "src", pkt.Src, "dest", pkt.Dest)

	targetPeer, err := s.RouteTable.Select(pkt.Dest)
	if err != nil {
		_ = income.Send(&Packet{
			Kind:   PacketKindClose,
			Src:    s.PeerID,
			Dest:   pkt.Src,
			DestID: pkt.SrcID,
			Error:  err.Error(),
		})
		log.Error(err, "choose")
		return err
	}
	if err := targetPeer.Send(pkt); err != nil {
		_ = income.Send(&Packet{
			Kind:   PacketKindClose,
			Src:    s.PeerID,
			Dest:   pkt.Src,
			DestID: pkt.SrcID,
			Error:  err.Error(),
		})
		log.Error(err, "forward")
		return err
	}
	return nil
}

func (s *PeerServer) localIn(channel IDChannel, pkt *Packet) {
	log.V(5).Info("local in", "pkt", pkt)

	switch pkt.Kind {
	case PacketKindOpen:
		go s.Connections.IncomeConn(channel, pkt)
	case PacketKindData:
		conn, err := s.Connections.GetConn(pkt.DestID)
		if err != nil || conn.closed {
			closePkt := &Packet{
				Kind:   PacketKindClose,
				SrcID:  pkt.DestID,
				Src:    s.PeerID,
				Dest:   pkt.Src,
				DestID: pkt.SrcID,
			}
			_ = channel.Send(closePkt)
			return
		}
		select {
		case conn.ack <- &connectData{remoteID: pkt.SrcID, err: pkt.Error, data: pkt.Data}:
		default:
			log.Error(errors.New("channel full"), "drop packet",
				"cid", conn.localConnectionID,
				"remote", conn.remotePeer,
				"remote cid", conn.remoteConnectionID,
			)
		}

	case PacketKindClose:
		go s.Connections.RemoveConn(pkt.DestID)
	case PacketKindRoute:
		go s.RouteTable.Update(pkt.Src, PacketDecode[PacketDataRoute](pkt.Data))
	}
}

type Dailer struct {
	PeerID string
	s      *PeerServer
}

func (s *PeerServer) DialerOnPeer(dest string) Dailer {
	return Dailer{s: s, PeerID: dest}
}

func (d Dailer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return d.s.Connections.OpenConnOn(network, address, timeout, d.PeerID)
}
