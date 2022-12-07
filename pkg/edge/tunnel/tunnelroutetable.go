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
	Channel     *ConnectedTunnel // channel is the direct connected channel
	Annotations map[string]string
	Children    map[string]Annotations // children are channels connected to the direct channel
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
	// default out tunnel
	if tun.Options.IsDefaultOut {
		t.defaultout = tun
	}
	t.OnChange(tun, data)
}

func (t *RouteTable) RouteExchange(idchannel *ConnectedTunnel, annotationsToSend Annotations) (*PacketDataRoute, error) {
	idchannel.AnnotationsSent = annotationsToSend
	if err := t.advertiseInit(idchannel, annotationsToSend); err != nil {
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
	log.Info("route exchange", "src", routepkt.Src, "data", routedata)
	return &routedata, nil
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
		Kind:  RouteUpdateKindOffline,
		Peers: removedPeers,
	})
}

func (t *RouteTable) OnChange(from *ConnectedTunnel, data PacketDataRoute) {
	id := from.ID
	log.Info("route changes", "src", id, "kind", data.Kind)
	t.mu.Lock()

	changeddata := PacketDataRoute{
		Kind:        data.Kind,
		Annotations: data.Annotations,
		Peers:       map[string]Annotations{},
	}

	switch data.Kind {
	case RouteUpdateKindReferesh:
		if data.Peers == nil {
			data.Peers = map[string]Annotations{}
		}
		if val, ok := t.records[id]; !ok {
			t.records[id] = &ChannelWithChildren{Channel: from, Annotations: data.Annotations, Children: data.Peers}
		} else {
			// may not happen
			val.Annotations = data.Annotations
			val.Children = data.Peers
			// refresh tunnel annotations
			val.Annotations = data.Annotations
		}
		changeddata.Kind = RouteUpdateKindOnline
		maps.Copy(changeddata.Peers, data.Peers)
		// advertise all peers and tun self are online
		changeddata.Peers[id] = data.Annotations
	case RouteUpdateKindKeepAlive:
		val, ok := t.records[id]
		if !ok {
			return
		}
		for alive, anno := range data.Peers {
			if _, ok := val.Children[alive]; ok {
				continue
			}
			val.Children[alive] = anno
		}
		// refresh tunnel annotations
		val.Annotations = data.Annotations
		changeddata.Kind = RouteUpdateKindKeepAlive
		maps.Copy(changeddata.Peers, data.Peers)
		// advertise all peers and tun self are keep alived
		changeddata.Peers[id] = data.Annotations
	case RouteUpdateKindOnline:
		val, ok := t.records[id]
		if !ok {
			return
		}
		for add, anno := range data.Peers {
			if _, ok := val.Children[add]; ok {
				continue
			}
			val.Children[add] = anno
		}
		changeddata.Kind = RouteUpdateKindOnline
		maps.Copy(changeddata.Peers, data.Peers)
	case RouteUpdateKindOffline:
		val, ok := t.records[id]
		if !ok {
			return
		}
		for remove := range data.Peers {
			if _, ok := val.Children[remove]; !ok {
				continue
			}
			delete(val.Children, remove)
		}
		changeddata.Annotations = val.Annotations
		changeddata.Kind = RouteUpdateKindOffline
		maps.Copy(changeddata.Peers, data.Peers)
	default:
		log.Info("unexpected route update", "data", data)
	}
	t.mu.Unlock()
	t.onchange(id, changeddata)
}

func (t *RouteTable) allRechablePeers(exclude string) map[string]Annotations {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ret := map[string]Annotations{}
	for k, val := range t.records {
		if k == exclude {
			continue
		}
		maps.Copy(ret, val.Children)
		ret[k] = val.Annotations
	}
	return ret
}

// advertise to other channel my updates expect the source channel
func (t *RouteTable) onchange(changesFrom string, changes PacketDataRoute) {
	t.advertise(changesFrom, changes)
}

func (t *RouteTable) advertise(changesFrom string, changes PacketDataRoute) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.s.eventer != nil {
		switch changes.Kind {
		case RouteUpdateKindOnline:
			t.s.eventer.sendWatcherEvent(TunnelEvent{
				From:            changesFrom,
				FromAnnotations: changes.Annotations,
				Kind:            EventKindConnected,
				Peers:           changes.Peers,
			})
		case RouteUpdateKindKeepAlive:
			t.s.eventer.sendWatcherEvent(TunnelEvent{
				From:            changesFrom,
				FromAnnotations: changes.Annotations,
				Kind:            EventKindKeepalive,
				Peers:           changes.Peers,
			})
		case RouteUpdateKindOffline:
			t.s.eventer.sendWatcherEvent(TunnelEvent{
				From:            changesFrom,
				FromAnnotations: changes.Annotations,
				Kind:            EventKindDisConnected,
				Peers:           changes.Peers,
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
		log.Info("advertise", "to", peer.Channel.ID, "data", changes)
		peer.Channel.Send(&Packet{
			Kind: PacketKindRoute,
			Dest: peerid,
			Src:  t.s.id,
			Data: PacketEncode(PacketDataRoute{
				Kind:        changes.Kind,
				Annotations: peer.Channel.AnnotationsSent,
				Peers:       changes.Peers,
			}),
		})
	}
}

func (t *RouteTable) Exists(id string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.records[id]
	return ok
}

func (t *RouteTable) SendKeepAlive(annotationsToSend Annotations) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, tun := range t.records {
		if !tun.Channel.Options.SendRouteChange {
			continue
		}
		t.advertiseKeepalive(tun.Channel, annotationsToSend)
	}
	return nil
}

// send all upstream init routes
func (t *RouteTable) advertiseKeepalive(idchannel *ConnectedTunnel, annotationsToSend Annotations) error {
	if !idchannel.Options.SendRouteChange {
		return nil
	}
	data := PacketDataRoute{
		Kind:        RouteUpdateKindKeepAlive,
		Annotations: annotationsToSend,
		Peers:       t.allRechablePeers(idchannel.ID),
	}
	log.Info("route keepalive", "dest", idchannel.ID, "data", data)
	// advetise self peers
	return idchannel.Send(&Packet{
		Kind: PacketKindRoute,
		Src:  t.s.id,
		Dest: idchannel.ID,
		Data: PacketEncode(data),
	})
}

// send all upstream init routes
func (t *RouteTable) advertiseInit(idchannel *ConnectedTunnel, annotationsToSend Annotations) error {
	data := PacketDataRoute{
		Kind:        RouteUpdateKindReferesh,
		Annotations: annotationsToSend,
	}
	if idchannel.Options.SendRouteChange {
		data.Peers = t.allRechablePeers(idchannel.ID)
	}
	log.Info("route init", "dest", idchannel.ID, "data", data)
	// advetise self peers
	return idchannel.Send(&Packet{
		Kind: PacketKindRoute,
		Src:  t.s.id,
		Dest: idchannel.ID,
		Data: PacketEncode(data),
	})
}
