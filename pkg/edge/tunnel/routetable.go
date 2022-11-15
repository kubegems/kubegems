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
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/exp/maps"
	"kubegems.io/kubegems/pkg/log"
)

type RouteTable struct {
	s        *PeerServer
	peers    map[string]PeerWithSubPeers
	mu       sync.RWMutex
	watchers sync.Map
}

func NewEmptyRouteTable(s *PeerServer) *RouteTable {
	return &RouteTable{
		s:     s,
		peers: map[string]PeerWithSubPeers{},
	}
}

type PeerWithSubPeers struct {
	Stream IDChannel
	Peers  map[string]struct{}
}

type PeerRule map[string]PeerRule

func (t *RouteTable) allRechablePeers() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ret := make([]string, len(t.peers))
	for k, val := range t.peers {
		ret = append(ret, k)
		ret = append(ret, maps.Keys(val.Peers)...)
	}
	sort.Strings(ret)
	return ret
}

func (t *RouteTable) Select(dest string) (Channel, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	log.Info("select", "table", t.peers)
	val, ok := t.peers[dest]
	if ok {
		return val.Stream, nil
	}
	for _, val := range t.peers {
		if _, ok := val.Peers[dest]; ok {
			return val.Stream, nil
		}
	}
	return nil, fmt.Errorf("no destination for peer %s", dest)
}

func (t *RouteTable) Connect(stream IDChannel, subpeers []string) {
	log.Info("peer connected", "peer", stream.ID)
	peermap := map[string]struct{}{}
	for _, v := range subpeers {
		peermap[v] = struct{}{}
	}

	t.mu.Lock()
	t.peers[stream.ID] = PeerWithSubPeers{Stream: stream, Peers: peermap}
	t.mu.Unlock()

	// advertise we add peers we direct connect and it's subpeers.
	t.advertisePeers(stream.ID, PacketDataRoute{
		Kind:     PeerUpdateKindAdd,
		SubPeers: append(subpeers, stream.ID),
	})
}

func (t *RouteTable) Disconnect(stream IDChannel) {
	log.Info("peer disconnected", "peer", stream.ID)

	t.mu.Lock()
	val, ok := t.peers[stream.ID]
	if !ok {
		return
	}
	removedPeers := append(maps.Keys(val.Peers), stream.ID)
	delete(t.peers, stream.ID)
	t.mu.Unlock()

	// advertise we remove peers we direct connect and it's subpeers.
	t.advertisePeers(stream.ID, PacketDataRoute{
		Kind:     PeerUpdateKindRemove,
		SubPeers: removedPeers,
	})
}

func (t *RouteTable) Update(id string, data PacketDataRoute) {
	log.Info("peer table update", "src", id, "data", data)
	t.mu.Lock()
	val, ok := t.peers[id]
	if !ok {
		return
	}
	changed := []string{}
	switch data.Kind {
	case PeerUpdateKindAdd:
		for _, add := range data.SubPeers {
			if _, ok := val.Peers[add]; ok {
				continue
			}
			val.Peers[add] = struct{}{}
			changed = append(changed, add)
		}
	case PeerUpdateKindRemove:
		for _, remove := range data.SubPeers {
			if _, ok := val.Peers[remove]; !ok {
				continue
			}
			delete(val.Peers, remove)
			changed = append(changed, remove)
		}
	case PeerUpdateKindRefresh:
		newmap := map[string]struct{}{}
		for _, add := range data.SubPeers {
			newmap[add] = struct{}{}
		}
		val.Peers = newmap
		changed = data.SubPeers
	}
	t.mu.Unlock()

	t.advertisePeers(id, PacketDataRoute{
		Kind:     data.Kind,
		SubPeers: changed,
	})
}

// advertise to other channel my updates expect the source channel
func (t *RouteTable) advertisePeers(changesFrom string, changes PacketDataRoute) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	t.sendWatcherEvent(RouteTableEvent{
		Peer:            changesFrom,
		PacketDataRoute: changes,
	})

	for peerid, peer := range t.peers {
		if peerid == changesFrom {
			continue
		}
		if peer.Stream.Options.ClientOnly {
			continue
		}
		data := changes
		log.Info("advertise", "to", peer.Stream.ID, "data", data)
		pkt := &Packet{
			Kind: PacketKindRoute,
			Dest: peerid,
			Src:  t.s.PeerID,
			Data: PacketEncode(data),
		}
		peer.Stream.Send(pkt)
	}
}

type RouteTableEvent struct {
	Peer string
	PacketDataRoute
}

type RouteTableWatcher struct {
	uid    string
	ch     chan RouteTableEvent
	cancel context.CancelFunc
}

func (r RouteTableWatcher) Result() <-chan RouteTableEvent {
	return r.ch
}

func (t *RouteTable) sendWatcherEvent(event RouteTableEvent) {
	t.watchers.Range(func(key, value any) bool {
		watcher, ok := value.(RouteTableWatcher)
		if !ok {
			return true
		}
		select {
		case watcher.ch <- event:
		default:
			//  full chan
		}
		return true
	})
}

func (t *RouteTableWatcher) Close() {
	t.cancel()
}

func (t *RouteTable) Watch(ctx context.Context) RouteTableWatcher {
	uid := uuid.NewString()

	ctx, cancel := context.WithCancel(ctx)

	watcher := RouteTableWatcher{
		uid:    uid,
		ch:     make(chan RouteTableEvent, 1),
		cancel: cancel,
	}

	log.Info("watcher start", "uid", uid)

	// send list
	watcher.ch <- RouteTableEvent{
		Peer: t.s.PeerID,
		PacketDataRoute: PacketDataRoute{
			Kind:     PeerUpdateKindRefresh,
			SubPeers: t.allRechablePeers(),
		},
	}
	// send watch
	t.watchers.Store(uid, watcher)

	go func() {
		<-ctx.Done()
		cancel()
		t.watchers.Delete(uid)
		log.Info("watcher exit", "uid", uid)
		close(watcher.ch)
	}()
	return watcher
}
