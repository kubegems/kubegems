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
	"fmt"
	"sync"

	"golang.org/x/exp/maps"
	"kubegems.io/kubegems/pkg/log"
)

type RouteTable struct {
	mu         sync.RWMutex
	s          *TunnelServer
	records    map[string]*ChannelWithChildren
	defaultout *ConnectedTunnel
}

func NewEmptyRouteTable(s *TunnelServer) *RouteTable {
	return &RouteTable{
		s:       s,
		records: map[string]*ChannelWithChildren{},
	}
}

type ChannelWithChildren struct {
	Channel     *ConnectedTunnel       `json:"channel,omitempty"` // channel is the direct connected channel
	Annotations map[string]string      `json:"annotations,omitempty"`
	Children    map[string]Annotations `json:"children,omitempty"` // children are channels connected to the direct channel
}

func (t *RouteTable) Select(dest string) (*ConnectedTunnel, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	val, ok := t.records[dest]
	if ok {
		return val.Channel, nil
	}
	for _, val := range t.records {
		if _, ok := val.Children[dest]; ok {
			return val.Channel, nil
		}
	}
	// try default out
	if t.defaultout != nil {
		return t.defaultout, nil
	}
	return nil, fmt.Errorf("no destination for peer %s", dest)
}

func (t *RouteTable) Connect(tun *ConnectedTunnel, data PacketDataRoute) {
	log.Info("tunnel connected", "tunnel", tun.ID)
	t.mu.Lock()
	if data.Peers == nil {
		data.Peers = map[string]Annotations{}
	}
	t.records[tun.ID] = &ChannelWithChildren{
		Channel:     tun,
		Annotations: data.Annotations,
		Children:    data.Peers,
	}
	t.mu.Unlock()

	// default out tunnel
	if tun.Options.IsDefaultOut {
		t.defaultout = tun
	}

	changes := maps.Clone(data.Peers)
	changes[tun.ID] = data.Annotations

	// advertise we add peers we direct connect and it's subpeers.
	t.advertise(tun.ID, PacketDataRoute{
		Kind:  RouteUpdateKindAppend,
		Peers: changes,
	})
}

func (t *RouteTable) Disconnect(stream *ConnectedTunnel) {
	log.Info("tunnel disconnected", "tunnel", stream.ID)

	t.mu.Lock()
	val, ok := t.records[stream.ID]
	if !ok {
		return
	}
	removedPeers := maps.Clone(val.Children)
	removedPeers[stream.ID] = val.Annotations
	delete(t.records, stream.ID)
	t.mu.Unlock()

	// close all connections to removedPeers
	t.s.connections.tunnelClose(stream)

	// advertise we remove peers we direct connect and it's subpeers.
	t.advertise(stream.ID, PacketDataRoute{
		Kind:  RouteUpdateKindRemove,
		Peers: removedPeers,
	})
}

func (t *RouteTable) Update(id string, data PacketDataRoute) {
	log.Info("route changed", "src", id, "data", data)
	t.mu.Lock()
	val, ok := t.records[id]
	if !ok {
		return
	}
	changed := map[string]Annotations{}
	switch data.Kind {
	case RouteUpdateKindAppend:
		for add, anno := range data.Peers {
			if _, ok := val.Children[add]; ok {
				continue
			}
			val.Children[add] = anno
			changed[add] = anno
		}
	case RouteUpdateKindRemove:
		for remove, anno := range data.Peers {
			if _, ok := val.Children[remove]; !ok {
				continue
			}
			delete(val.Children, remove)
			changed[remove] = anno
		}
	default:
		log.Info("unexpected route update", "data", data)
	}
	log.Info("current route table", "records", t.records)
	t.mu.Unlock()

	t.advertise(id, PacketDataRoute{
		Kind:  data.Kind,
		Peers: changed,
	})
}

func (t *RouteTable) allRechablePeers() map[string]Annotations {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ret := map[string]Annotations{}
	for k, val := range t.records {
		maps.Copy(ret, val.Children)
		ret[k] = val.Annotations
	}
	return ret
}

// advertise to other channel my updates expect the source channel
func (t *RouteTable) advertise(changesFrom string, changes PacketDataRoute) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.s.eventer != nil {
		switch changes.Kind {
		case RouteUpdateKindAppend:
			t.s.eventer.sendWatcherEvent(TunnelEvent{
				Kind:  EventKindConnected,
				Peers: changes.Peers,
			})
		case RouteUpdateKindRemove:
			t.s.eventer.sendWatcherEvent(TunnelEvent{
				Kind:  EventKindDisConnected,
				Peers: changes.Peers,
			})
		}
	}

	for peerid, peer := range t.records {
		if peerid == changesFrom {
			continue
		}
		if !peer.Channel.Options.SendRouteChange {
			continue
		}
		data := changes
		log.Info("advertise", "to", peer.Channel.ID, "data", data)
		pkt := &Packet{
			Kind: PacketKindRoute,
			Dest: peerid,
			Src:  t.s.id,
			Data: PacketEncode(data),
		}
		peer.Channel.Send(pkt)
	}
}
