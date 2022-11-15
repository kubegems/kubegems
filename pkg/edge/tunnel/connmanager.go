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
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/log"
)

const MaxOpenConnectTimeout = 30 * time.Second

type ConnectionManager struct {
	s           *PeerServer
	mu          sync.RWMutex
	autoinc     int64
	connections map[int64]*TunnelConn
}

func NewConectionManager(s *PeerServer) *ConnectionManager {
	return &ConnectionManager{
		s:           s,
		connections: map[int64]*TunnelConn{},
	}
}

func (cm *ConnectionManager) IncomeConn(income Channel, pkt *Packet) {
	dialOptions := PacketDecode[PacketDataDialOptions](pkt.Data)
	localConnID := atomic.AddInt64(&cm.autoinc, 1)

	log.Info("dial options", "opts", dialOptions)
	conn, err := net.DialTimeout(dialOptions.Network, dialOptions.Address, dialOptions.Timeout)
	if err != nil {
		log.Error(err, "dial timeout", "options", dialOptions)
		_ = income.Send(&Packet{
			Kind:   PacketKindClose,
			Dest:   pkt.Src,
			DestID: pkt.SrcID,
			Error:  err.Error(),
		})
		return
	} else {
		_ = income.Send(&Packet{
			Kind:   PacketKindData,
			Dest:   pkt.Src,
			DestID: pkt.SrcID,
			SrcID:  localConnID,
			Data:   []byte{}, // send empty data to ack opened
		})
	}

	tunConn := &TunnelConn{
		cm:                 cm,
		channel:            income,
		localConnectionID:  localConnID,
		localPeer:          cm.s.PeerID,
		remoteConnectionID: pkt.SrcID,
		remotePeer:         pkt.Src,
		rawConn:            conn,
		ack:                make(chan *connectData, DataChannelSize),
	}

	cm.mu.Lock()
	cm.connections[localConnID] = tunConn
	cm.mu.Unlock()
	log.Info("opened connection",
		"cid", localConnID,
		"remote", tunConn.remotePeer,
		"remote cid", tunConn.remoteConnectionID)

	eg := errgroup.Group{}
	eg.Go(tunConn.remoteToTunnel)
	eg.Go(tunConn.tunnelToRemote)
	if err := eg.Wait(); err != nil {
		log.Error(err, "connection close", "cid", tunConn.localConnectionID)
	}
	cm.RemoveConn(tunConn.localConnectionID)
}

func (cm *ConnectionManager) OpenConnOn(network, address string, timeout time.Duration, destPeer string) (net.Conn, error) {
	dialctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tunconn, err := cm.prepareConn(destPeer)
	if err != nil {
		return nil, err
	}
	log.Info("openning connection",
		"peer", destPeer, "cid", tunconn.localConnectionID,
		"network", network, "address", address,
	)

	// send open pkt to dest
	openPkt := &Packet{
		Kind:  PacketKindOpen,
		SrcID: tunconn.localConnectionID,
		Src:   cm.s.PeerID,
		Dest:  destPeer,
		Data: PacketEncode(
			PacketDataDialOptions{
				Network: network,
				Address: address,
				Timeout: timeout,
			}),
	}
	if err := tunconn.channel.Send(openPkt); err != nil {
		return nil, err
	}

	// wait open ack
	select {
	case ack := <-tunconn.ack:
		if ack == nil {
			return nil, net.ErrClosed
		}
		if msg := ack.err; msg != "" {
			cm.RemoveConn(tunconn.localConnectionID)
			return nil, errors.New(msg)
		}
		if ack.remoteID == 0 {
			cm.RemoveConn(tunconn.localConnectionID)
			return nil, errors.New("empty remote connection id")
		}
		// opened
		tunconn.remoteConnectionID = ack.remoteID
		log.Info("connection opend",
			"network", network, "address", address,
			"cid", tunconn.localConnectionID,
			"remote", tunconn.remotePeer,
			"remote cid", tunconn.remoteConnectionID,
		)
		return tunconn, nil
	case <-dialctx.Done():
		cm.RemoveConn(tunconn.localConnectionID)
		return nil, dialctx.Err()
	case <-time.After(MaxOpenConnectTimeout):
		cm.RemoveConn(tunconn.localConnectionID)
		return nil, fmt.Errorf("dial timeout")
	}
}

func (cm *ConnectionManager) prepareConn(dest string) (*TunnelConn, error) {
	// find peer
	targetPeer, err := cm.s.RouteTable.Select(dest)
	if err != nil {
		return nil, err
	}
	tunconn := &TunnelConn{
		cm:                cm,
		channel:           targetPeer,
		localConnectionID: atomic.AddInt64(&cm.autoinc, 1),
		localPeer:         cm.s.PeerID,
		remotePeer:        dest,
		ack:               make(chan *connectData, DataChannelSize),
	}
	cm.mu.Lock()
	cm.connections[tunconn.localConnectionID] = tunconn
	cm.mu.Unlock()
	return tunconn, nil
}

func (cm *ConnectionManager) GetConn(connID int64) (*TunnelConn, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	val, ok := cm.connections[connID]
	if !ok {
		return nil, fmt.Errorf("no connection of id %d", connID)
	}
	return val, nil
}

func (cm *ConnectionManager) RemoveConn(connID int64) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn, ok := cm.connections[connID]; ok {
		log.Info("remove connection",
			"cid", connID,
			"remote", conn.remotePeer,
			"remote cid", conn.remoteConnectionID,
		)
		conn.cm = nil
		conn.channel = nil
		close(conn.ack)
		conn.closed = true
		if conn.rawConn != nil {
			conn.rawConn.Close()
		}
		delete(cm.connections, connID)
	}
	return nil
}
