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
	"errors"
	"io"
	"net"
	"time"

	"kubegems.io/kubegems/pkg/log"
)

const DataChannelSize = 512

type TunnelConn struct {
	channel            Channel
	cm                 *ConnectionManager
	localConnectionID  int64
	localPeer          string
	remoteConnectionID int64
	remotePeer         string
	rdata              []byte
	rawConn            net.Conn
	closed             bool
	ack                chan *connectData
}

func (c *TunnelConn) remoteToTunnel() error {
	var buf [1 << 12]byte
	for {
		n, err := c.rawConn.Read(buf[:])
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := c.Write(buf[:n]); err != nil {
			return err
		}
	}
}

func (c *TunnelConn) tunnelToRemote() error {
	for ack := range c.ack {
		if ack == nil /*channel closed*/ {
			return net.ErrClosed
		}
		if ack.err != "" {
			return errors.New(ack.err)
		}
		pos := 0
		for {
			n, err := c.rawConn.Write(ack.data[pos:])
			if err != nil {
				return err
			}
			if n > 0 {
				pos += n
			}
		}
	}
	return nil
}

type connectData struct {
	remoteID int64
	err      string
	data     []byte
}

func (c *TunnelConn) Read(b []byte) (n int, err error) {
	var data []byte
	if c.rdata != nil {
		data = c.rdata
	} else {
		ack := <-c.ack
		if ack == nil /*closed*/ {
			return 0, net.ErrClosed
		}
		if ack.err != "" {
			return 0, errors.New(ack.err)
		}
		data = ack.data
	}
	if data == nil {
		return 0, io.EOF
	}
	if len(data) > len(b) {
		copy(b, data[:len(b)])
		c.rdata = data[len(b):]
		return len(b), nil
	}
	c.rdata = nil
	copy(b, data)
	return len(data), nil
}

func (c *TunnelConn) Write(b []byte) (n int, err error) {
	if c.closed {
		return 0, net.ErrClosed
	}
	req := &Packet{
		Kind:   PacketKindData,
		Src:    c.localPeer,
		SrcID:  c.localConnectionID,
		Dest:   c.remotePeer,
		DestID: c.remoteConnectionID,
		Data:   b,
	}
	if err = c.channel.Send(req); err != nil {
		return 0, err
	}
	return len(b), err
}

func (c *TunnelConn) Close() error {
	req := &Packet{
		Kind:   PacketKindClose,
		Src:    c.localPeer,
		SrcID:  c.localConnectionID,
		Dest:   c.remotePeer,
		DestID: c.remoteConnectionID,
	}
	defer c.cm.RemoveConn(c.localConnectionID)
	log.Info("close connection", "cid", c.localConnectionID)
	return c.channel.Send(req)
}

func (c *TunnelConn) LocalAddr() net.Addr {
	return nil
}

func (c *TunnelConn) RemoteAddr() net.Addr {
	return nil
}

func (c *TunnelConn) SetDeadline(t time.Time) error {
	panic("not implemented") // TODO: Implement
}

func (c *TunnelConn) SetReadDeadline(t time.Time) error {
	panic("not implemented") // TODO: Implement
}

func (c *TunnelConn) SetWriteDeadline(t time.Time) error {
	panic("not implemented") // TODO: Implement
}
