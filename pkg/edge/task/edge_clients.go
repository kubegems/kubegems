// Copyright 2023 The kubegems.io Authors
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

package task

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	edgeclient "kubegems.io/kubegems/pkg/edge/client"
	"kubegems.io/kubegems/pkg/utils/kube"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const DefaultEdgeResourceQueueSize = 1024

type EdgeClusterResourceChangedCallback func(uid string, obj client.Object)

type EdgeClientsHolder struct {
	//nolint: containedctx
	basectx context.Context
	server  string
	events  chan EdgeClusterEvent
	clients map[string]clientWithCancel
	mu      sync.RWMutex
}

type clientWithCancel struct {
	cli  client.Client
	stop context.CancelFunc
}

type EdgeClusterEvent struct {
	UID           string
	TaskName      string
	TaskNamespace string
}

func NewEdgeClientsHolder(ctx context.Context, server string) (*EdgeClientsHolder, error) {
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		return nil, fmt.Errorf("scheme is required in edge server address")
	}
	return &EdgeClientsHolder{
		basectx: ctx,
		server:  server,
		clients: map[string]clientWithCancel{},
		events:  make(chan EdgeClusterEvent, DefaultEdgeResourceQueueSize),
	}, nil
}

func (c *EdgeClientsHolder) Invalid(uid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cancelcli, ok := c.clients[uid]
	if ok {
		cancelcli.stop()
		delete(c.clients, uid)
	}
}

func (c *EdgeClientsHolder) Get(uid string) (client.Client, error) {
	c.mu.RLock()
	cancelcli, ok := c.clients[uid]
	c.mu.RUnlock()
	if ok {
		return cancelcli.cli, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if cli, ok := c.clients[uid]; ok {
		return cli.cli, nil
	}
	cli, err := edgeclient.NewEdgeClient(c.server, uid)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(c.basectx)
	cancelcli = clientWithCancel{cli: kube.NewCachedClient(ctx, cli, c.eventhandler(cli, uid)), stop: cancel}
	c.clients[uid] = cancelcli
	return cancelcli.cli, nil
}

func (c *EdgeClientsHolder) eventhandler(cli client.Client, cluster string) cache.ResourceEventHandler {
	enqueue := func(data any) {
		obj, ok := data.(client.Object)
		if !ok {
			return
		}
		ownerobj, _ := FindOwnerControllerRecursively(c.basectx, cli, obj)
		if ownerobj == nil {
			return
		}
		// filter obj has edge task annotation
		taskname, taskns := ExtractEdgeTask(ownerobj)
		if taskname == "" || taskns == "" {
			return
		}
		select {
		// nolint: forcetypeassert
		case c.events <- EdgeClusterEvent{UID: cluster, TaskName: taskname, TaskNamespace: taskns}:
		default:
			logr.FromContextOrDiscard(c.basectx).Info("edge resource event queue is full, drop event", "uid", cluster, "obj", obj)
		}
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    enqueue,
		UpdateFunc: func(oldObj, newObj interface{}) { enqueue(newObj) },
		DeleteFunc: enqueue,
	}
}

// Start behaves like controller-runtime's source.Source interface
func (c *EdgeClientsHolder) SourceFunc(ctx context.Context, cli client.Client) source.Func {
	// it's a producer to reconciler
	return func(_ context.Context, _ handler.EventHandler, queue workqueue.RateLimitingInterface, _ ...predicate.Predicate) error {
		go func() {
			log := logr.FromContextOrDiscard(ctx)
			log.Info("edge resource event queue started")
			defer log.Info("edge resource event queue stopped")
			for {
				select {
				case <-ctx.Done():
					return
				case event := <-c.events:
					taskname, tasknamespace := event.TaskName, event.TaskNamespace
					log.Info("trigger reconcile", "cluster", event.UID, "task", taskname, "namespace", tasknamespace)
					// chekc task spec.edgeclustername
					queue.Add(ctrl.Request{NamespacedName: client.ObjectKey{Name: taskname, Namespace: tasknamespace}})
				}
			}
		}()
		return nil
	}
}
