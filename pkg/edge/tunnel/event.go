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
