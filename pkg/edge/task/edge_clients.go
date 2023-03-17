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
	clients sync.Map
	events  chan EdgeClusterEvent
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
		events:  make(chan EdgeClusterEvent, DefaultEdgeResourceQueueSize),
	}, nil
}

func (c *EdgeClientsHolder) Get(uid string) (client.Client, error) {
	if cli, ok := c.clients.Load(uid); ok {
		// nolint: forcetypeassert
		return cli.(client.Client), nil
	}
	cli, err := edgeclient.NewEdgeClient(c.server, uid)
	if err != nil {
		return nil, err
	}
	cli = kube.NewCachedClient(c.basectx, cli, c.eventhandler(cli, uid))
	c.clients.Store(uid, cli)
	return cli, nil
}

func (c *EdgeClientsHolder) eventhandler(cli client.Client, cluster string) cache.ResourceEventHandler {
	enqueue := func(data any) {
		obj, ok := data.(client.Object)
		if !ok {
			return
		}
		ownerobj, err := FindOwnerControllerRecursively(c.basectx, cli, obj)
		if err != nil {
			logr.FromContextOrDiscard(c.basectx).Error(err, "failed to find owner controller",
				"cluster", cluster,
				"gvk", obj.GetObjectKind().GroupVersionKind().String(),
				"name", client.ObjectKeyFromObject(obj).String(),
			)
			return
		}
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
func (c *EdgeClientsHolder) SourceFunc(cli client.Client) source.Func {
	// it's a producer to reconciler
	return func(ctx context.Context, _ handler.EventHandler, queue workqueue.RateLimitingInterface, _ ...predicate.Predicate) error {
		go func() {
			logr.FromContextOrDiscard(ctx).Info("edge resource event queue started")
			defer logr.FromContextOrDiscard(ctx).Info("edge resource event queue stopped")
			for {
				select {
				case <-ctx.Done():
					return
				case event := <-c.events:
					taskname, tasknamespace := event.TaskName, event.TaskNamespace
					queue.Add(ctrl.Request{NamespacedName: client.ObjectKey{Name: taskname, Namespace: tasknamespace}})
				}
			}
		}()
		return nil
	}
}
