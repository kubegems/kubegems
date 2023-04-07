package task

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	edgeclient "kubegems.io/kubegems/pkg/edge/client"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	UID string
	Obj client.Object
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
	cli = kube.NewCachedClient(c.basectx, cli, c.eventhandler(uid))
	c.clients.Store(uid, cli)
	return cli, nil
}

func (c *EdgeClientsHolder) eventhandler(uid string) cache.ResourceEventHandler {
	enqueue := func(obj any) {
		select {
		// nolint: forcetypeassert
		case c.events <- EdgeClusterEvent{UID: uid, Obj: obj.(client.Object)}:
		default:
			logr.FromContextOrDiscard(c.basectx).Info("edge resource event queue is full, drop event", "uid", uid, "obj", obj)
		}
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    enqueue,
		UpdateFunc: func(oldObj, newObj interface{}) { enqueue(newObj) },
		DeleteFunc: enqueue,
	}
}

// Start behaves like controller-runtime's source.Source interface
func (c *EdgeClientsHolder) SourceFunc() source.Func {
	defaultns := "kubegems-edge"
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
					// resources change from edge cluster uid
					// trigger the task of name uid to reconcile
					// TODO: we can find target edge task from resource's annotations instead of use uid as task name
					uid := event.UID
					queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{Name: uid, Namespace: defaultns}})
				}
			}
		}()
		return nil
	}
}
