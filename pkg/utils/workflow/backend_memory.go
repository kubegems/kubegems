// Copyright 2024 The kubegems.io Authors
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

package workflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"kubegems.io/kubegems/pkg/log"
)

var _ Backend = &InmemoryBackend{}

type kv struct {
	key        string
	val        []byte
	createTime time.Time
	expireTime time.Time
}

type kvwatcher struct {
	key string
	fn  OnChangeFunc
}

func NewInmemoryBackend(ctx context.Context) *InmemoryBackend {
	backend := &InmemoryBackend{
		db:         make(map[string]kv),
		subeventch: make(chan kv, 64),
		subs:       make(map[string]OnChangeFunc),
		watchch:    make(chan kv, 64),
		watchers:   make(map[string]kvwatcher),
	}
	go backend.run(ctx)
	return backend
}

// worker 仅允许一个实例启动，并且队列中的任务仅存在内存中，不支持持久化。
type InmemoryBackend struct {
	db     map[string]kv
	dblock sync.RWMutex

	sublock    sync.RWMutex
	subs       map[string]OnChangeFunc
	subeventch chan kv

	watchch   chan kv
	watchlock sync.RWMutex
	watchers  map[string]kvwatcher
}

// nolint: gocognit
func (t *InmemoryBackend) run(ctx context.Context) error {
	concurrency := 3
	log := logr.FromContextOrDiscard(ctx).WithName("inmemorybackend")
	log.Info("start inmemory backend", "concurrency", concurrency)
	for i := 0; i < concurrency; i++ {
		// watcher
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case kv := <-t.watchch:
					for _, watcher := range t.watchers {
						if strings.HasPrefix(kv.key, watcher.key) {
							log.V(5).Info("watcher nofity", "key", kv.key)
							watcher.fn(ctx, kv.key, kv.val)
						}
					}
				}
			}
		}()

		// subevent
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case kv := <-t.subeventch:
					for _, onchange := range t.subs {
						log.V(5).Info("subscriber nofity", "key", kv.key)
						onchange(ctx, kv.key, kv.val)
					}
				}
			}
		}()
	}
	go func() {
		duration := 1 * time.Minute
		log.V(5).Info("start expire worker", "duration", duration)
		timer := time.NewTimer(duration)
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				t.removeExpired(ctx)
				timer.Reset(duration)
			}
		}
	}()
	return nil
}

func (t *InmemoryBackend) removeExpired(ctx context.Context) {
	log := logr.FromContextOrDiscard(ctx)
	log.V(5).Info("start remove expired")
	t.dblock.Lock()
	defer t.dblock.Unlock()

	now := time.Now()
	for k, v := range t.db {
		if v.expireTime.Before(now) {
			log.Info("remove expire", "key", k)
			delete(t.db, k)
			t.event(kv{key: k})
		}
	}
}

func (t *InmemoryBackend) event(kv kv) {
	log.Info("dispatch event", "key", kv.key)
	select {
	case t.watchch <- kv:
	default:
		log.Info("event channel full", "key", kv.key)
	}
}

// Del implements Backend.
func (t *InmemoryBackend) Del(ctx context.Context, key string) error {
	logr.FromContextOrDiscard(ctx).V(5).Info("del", "key", key)
	t.dblock.Lock()
	defer t.dblock.Unlock()
	delete(t.db, key)
	return nil
}

// Get implements Backend.
func (t *InmemoryBackend) Get(ctx context.Context, key string) ([]byte, error) {
	logr.FromContextOrDiscard(ctx).V(5).Info("get", "key", key)
	t.dblock.RLock()
	defer t.dblock.RUnlock()
	return t.db[key].val, nil
}

// List implements Backend.
func (t *InmemoryBackend) List(ctx context.Context, keyprefix string) (map[string][]byte, error) {
	logr.FromContextOrDiscard(ctx).V(5).Info("list", "keyprefix", keyprefix)
	ret := make(map[string][]byte)

	t.dblock.RLock()
	defer t.dblock.RUnlock()
	for k, v := range t.db {
		if strings.HasPrefix(k, keyprefix) {
			ret[k] = v.val
		}
	}
	return ret, nil
}

// Pub implements Backend.
func (t *InmemoryBackend) Pub(ctx context.Context, name string, key string, val []byte) error {
	logr.FromContextOrDiscard(ctx).V(5).Info("pub", "name", name, "key", key, "val", string(val))
	select {
	case t.subeventch <- kv{key: key, val: val}:
	case <-ctx.Done():
	default:
		return fmt.Errorf("subevent channel full")
	}
	return nil
}

// Put implements Backend.
func (t *InmemoryBackend) Put(ctx context.Context, key string, val []byte, ttl ...time.Duration) error {
	logr.FromContextOrDiscard(ctx).V(5).Info("put", "key", key, "val", string(val))
	kv := kv{
		key:        key,
		val:        val,
		createTime: time.Now(),
	}
	if len(ttl) > 0 {
		kv.expireTime = kv.createTime.Add(ttl[0])
	}
	t.dblock.Lock()
	t.db[key] = kv
	t.dblock.Unlock()

	t.event(kv)
	return nil
}

// 这里的sub要求多个消费者共享同一个topic下的数据，且无重复。
func (t *InmemoryBackend) Sub(ctx context.Context, name string, onchange OnChangeFunc, opts ...SubOption) error {
	options := &SubOptions{Concurrency: 1}
	for _, opt := range opts {
		opt(options)
	}
	uid := uuid.New().String()
	logr.FromContextOrDiscard(ctx).V(5).Info("sub", "name", name, "uid", uid, "concurrency", options.Concurrency)

	concurrency := make(chan struct{}, options.Concurrency)

	t.sublock.Lock()
	t.subs[uid] = func(ctx context.Context, key string, val []byte) error {
		concurrency <- struct{}{}
		go func() {
			defer func() {
				<-concurrency
			}()
			onchange(ctx, key, val)
		}()
		return nil
	}
	t.sublock.Unlock()
	defer func() {
		logr.FromContextOrDiscard(ctx).V(5).Info("unsub", "name", name, "uid", uid)
		t.sublock.Lock()
		delete(t.subs, uid)
		t.sublock.Unlock()
	}()
	<-ctx.Done()
	return nil
}

// Watch implements Backend.
func (t *InmemoryBackend) Watch(ctx context.Context, key string, onchange OnChangeFunc) error {
	uuid := uuid.New().String()
	logr.FromContextOrDiscard(ctx).V(5).Info("watch", "key", key, "uuid", uuid)
	t.watchlock.Lock()
	t.watchers[uuid] = kvwatcher{key: key, fn: onchange}
	t.watchlock.Unlock()
	defer func() {
		logr.FromContextOrDiscard(ctx).V(5).Info("unwatch", "key", key, "uuid", uuid)
		t.watchlock.Lock()
		delete(t.watchers, uuid)
		t.watchlock.Unlock()
	}()
	<-ctx.Done()
	return nil
}
