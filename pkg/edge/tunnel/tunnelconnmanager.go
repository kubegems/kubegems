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

	"kubegems.io/kubegems/pkg/log"
)

const (
	DefaultDataChannelSize = 512
	MaxOpenConnectTimeout  = 30 * time.Second
)

type Connections struct {
	local       string
	autoinc     int64
	mu          sync.RWMutex
	connections map[int64]*TunnelConn // localcid -> tunnel
}

func (c *Connections) pending(tun *ConnectedTunnel, remote string, remotecid int64) *TunnelConn {
	tunconn := &TunnelConn{
		c:                  c,
		channel:            tun,
		remote:             remote,
		remoteConnectionID: remotecid,
		local:              c.local,
		localConnectionID:  atomic.AddInt64(&c.autoinc, 1),
		ack:                make(chan *connectData, DefaultDataChannelSize),
		closed:             false,
	}
	c.mu.Lock()
	c.connections[tunconn.localConnectionID] = tunconn
	c.mu.Unlock()
	return tunconn
}

func (c *Connections) accepted(tun *TunnelConn, rawconn net.Conn) {
	tun.rawConn = rawconn
}

func (c *Connections) opened(tun *TunnelConn, remotecid int64) {
	tun.remoteConnectionID = remotecid
}

func (c *Connections) get(localcid int64) *TunnelConn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.connections[localcid]
	if !ok {
		return nil
	}
	return val
}

func (c *Connections) close(localcid int64) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeWihoutLock(localcid)
}

func (c *Connections) closeWihoutLock(localcid int64) (err error) {
	if conn, ok := c.connections[localcid]; ok {
		log.Info("connection closed",
			"cid", conn.localConnectionID,
			"remote", conn.remote,
			"remote cid", conn.remoteConnectionID,
		)
		close(conn.ack)
		conn.closed = true
		if conn.rawConn != nil {
			// https://man7.org/linux/man-pages/man2/close.2.html
			// close() will fail when a routine on block write()
			err = conn.rawConn.Close()
		}
		delete(c.connections, localcid)
	}
	return
}

func (c *Connections) flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for cid, conn := range c.connections {
		_ = conn.c.closeWihoutLock(cid)
	}
}

type ConnectionManager struct {
	s       *TunnelServer
	tunnels map[string]*Connections
}

func NewConectionManager(s *TunnelServer) *ConnectionManager {
	return &ConnectionManager{
		s:       s,
		tunnels: map[string]*Connections{},
	}
}

func (cm *ConnectionManager) tunnel(tun *ConnectedTunnel) *Connections {
	val, ok := cm.tunnels[tun.ID]
	if !ok {
		val = &Connections{
			connections: make(map[int64]*TunnelConn),
			local:       cm.s.id,
		}
		cm.tunnels[tun.ID] = val
	}
	return val
}

// func (cm *ConnectionManager) remote(remote string) *Connections {
// 	if val, ok := cm.hosts[remote]; ok {
// 		return val
// 	} else {
// 		val = &Connections{
// 			connections: map[int64]*TunnelConn{},
// 			remote:      remote,
// 			local:       cm.s.id,
// 		}
// 		cm.hosts[remote] = val
// 		return val
// 	}
// }

func (cm *ConnectionManager) Open(network, address string, timeout time.Duration, dest string) (conn net.Conn, err error) {
	dialctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// selecte a tunnel to dest
	connectedChannel, err := cm.s.routeTable.Select(dest)
	if err != nil {
		return nil, err
	}

	// send open pkt to dest
	tunconn := cm.tunnel(connectedChannel).pending(connectedChannel, dest, 0)
	log.Info("connection openning",
		"peer", dest, "cid", tunconn.localConnectionID,
		"network", network, "address", address,
	)
	if err := tunconn.sendOpen(PacketDataOpen{Network: network, Address: address, Timeout: timeout}); err != nil {
		return nil, err
	}
	// wait open ack
	select {
	case ack := <-tunconn.ack:
		if ack == nil {
			_ = tunconn.Close()
			return nil, net.ErrClosed
		}
		if msg := ack.err; msg != "" {
			_ = tunconn.Close()
			return nil, errors.New(msg)
		}
		if ack.remoteID == 0 {
			_ = tunconn.Close()
			return nil, errors.New("empty remote connection id")
		}
		// established
		tunconn.opened(ack.remoteID)
		log.Info("connection opend",
			"network", network, "address", address,
			"cid", tunconn.localConnectionID,
			"remote", tunconn.channel.ID,
			"remote cid", tunconn.remoteConnectionID,
		)
		return tunconn, nil
	case <-dialctx.Done():
		_ = tunconn.Close()
		return nil, dialctx.Err()
	case <-time.After(MaxOpenConnectTimeout):
		_ = tunconn.Close()
		return nil, fmt.Errorf("dial timeout")
	}
}

func (cm *ConnectionManager) accept(fromtunnel *ConnectedTunnel, remote string, remotecid int64, dialOptions PacketDataOpen) error {
	tunConn := cm.tunnel(fromtunnel).pending(fromtunnel, remote, remotecid)
	defer tunConn.Close()

	log := log.LogrLogger.WithValues(
		"local cid", tunConn.localConnectionID,
		"remote", tunConn.channel.ID,
		"remote cid", tunConn.remoteConnectionID)

	log.Info("dial options", "opts", dialOptions)
	conn, err := net.DialTimeout(dialOptions.Network, dialOptions.Address, dialOptions.Timeout)
	if err != nil {
		log.Error(err, "dial timeout", "options", dialOptions)
		_ = tunConn.sendClose(err)
		return err
	}
	if err := tunConn.sendData([]byte{}); err != nil {
		log.Error(err, "connection send ack")
		return err
	}
	// established
	log.Info("connection opened")
	defer func() {
		log.Info("connection routine exit")
	}()
	if err := tunConn.accepted(conn); err != nil {
		log.Error(err, "connection exit")
		return err
	}
	return nil
}

func (cm *ConnectionManager) ack(fromtunnel *ConnectedTunnel, from string, fromCID int64, localcid int64, data []byte, err string) error {
	log.Info("packet ack", "cid", localcid, "remote", from, "remote cid", fromCID)
	conn := cm.tunnel(fromtunnel).get(localcid)
	if conn == nil || conn.closed {
		return net.ErrClosed
	}
	select {
	case conn.ack <- &connectData{remoteID: fromCID, err: err, data: data}:
	default:
		log.Error(errors.New("channel full"), "drop packet",
			"cid", conn.localConnectionID,
			"remote", conn.channel.ID,
			"remote cid", conn.remoteConnectionID,
		)
	}
	return nil
}

func (cm *ConnectionManager) close(fromtunnel *ConnectedTunnel, remote string, remotecid int64, localcid int64) (err error) {
	return cm.tunnel(fromtunnel).close(localcid)
}

// remoteLost close all connections on remote
func (cm *ConnectionManager) tunnelClose(tunnel *ConnectedTunnel) {
	log.Info("tunnel lost", "id", tunnel.ID)
	cm.tunnel(tunnel).flush()
}
