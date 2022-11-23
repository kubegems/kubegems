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
	"sync"

	"github.com/google/uuid"
	"kubegems.io/kubegems/pkg/log"
)

type TunnelEventer struct {
	watchers sync.Map
	s        *TunnelServer
}

func NewTunnelEventer(s *TunnelServer) *TunnelEventer {
	return &TunnelEventer{s: s}
}

type EventKind string

const (
	EventKindConnected    EventKind = "connected"
	EventKindDisConnected EventKind = "disconnected"
)

type TunnelEvent struct {
	Kind  EventKind
	Peers map[string]Annotations
}

type EventWatcher struct {
	uid    string
	ch     chan TunnelEvent
	cancel context.CancelFunc
}

func (r EventWatcher) Result() <-chan TunnelEvent {
	return r.ch
}

func (t *TunnelEventer) sendWatcherEvent(event TunnelEvent) {
	t.watchers.Range(func(key, value any) bool {
		watcher, ok := value.(EventWatcher)
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

func (t *EventWatcher) Close() {
	t.cancel()
}

func (t *TunnelEventer) Watch(ctx context.Context) EventWatcher {
	ctx, cancel := context.WithCancel(ctx)
	uid := uuid.NewString()
	watcher := EventWatcher{
		uid:    uid,
		ch:     make(chan TunnelEvent, 1),
		cancel: cancel,
	}
	log.Info("watcher start", "uid", uid)

	// add to watch to receive changes
	t.watchers.Store(uid, watcher)
	// send init list
	watcher.ch <- TunnelEvent{
		Kind:  EventKindConnected,
		Peers: t.s.routeTable.allRechablePeers(),
	}
	go func() {
		<-ctx.Done()
		cancel()
		t.watchers.Delete(uid)
		log.Info("watcher exit", "uid", uid)
		close(watcher.ch)
	}()
	return watcher
}
