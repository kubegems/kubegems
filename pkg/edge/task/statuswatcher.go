package task

import (
	"context"
	"errors"
	"sync"

	"github.com/go-logr/logr"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	edgev1beta1 "kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// nolint: containedctx
type EdgeStatusWatcher struct {
	basectx  context.Context
	clients  *EdgeClientsHolder
	clusters map[string]*perClientWatcher
	mu       sync.Mutex
}

func NewEdgeStatusWatcher(basectx context.Context, clients *EdgeClientsHolder) *EdgeStatusWatcher {
	return &EdgeStatusWatcher{
		clients:  clients,
		basectx:  basectx,
		clusters: map[string]*perClientWatcher{},
	}
}

func (w *EdgeStatusWatcher) StartWatch(edgetask *edgev1beta1.EdgeTask, onchange func(obj client.Object) bool) {
	log := logr.FromContextOrDiscard(w.basectx).WithValues("edgetask", edgetask.Name)

	w.mu.Lock()
	defer w.mu.Unlock()

	uid := edgetask.Name

	cluster, ok := w.clusters[uid]
	if !ok {
		cli, err := w.clients.Get(uid)
		if err != nil {
			log.Error(err, "failed to get edge client")
			return
		}
		watchcli, ok := cli.(client.WithWatch)
		if !ok {
			log.Error(err, "edge client does not support watch")
			return
		}
		cluster = &perClientWatcher{
			basectx:  w.basectx,
			cli:      watchcli,
			watchers: make(map[schema.GroupVersionKind]watch.Interface),
		}
		w.clusters[uid] = cluster
	}
	cluster.SetWatch(edgetask, onchange)
}

func (w *EdgeStatusWatcher) StopWatch(edgetask *edgev1beta1.EdgeTask) {}

// nolint: containedctx
type perClientWatcher struct {
	basectx    context.Context
	cli        client.WithWatch
	watchers   map[schema.GroupVersionKind]watch.Interface
	watchersmu sync.Mutex
}

type taskwithcallback struct {
	task     *edgev1beta1.EdgeTask
	callback func(obj client.Object) bool
}

func (c *perClientWatcher) SetWatch(edgetask *edgev1beta1.EdgeTask, callback func(obj client.Object) bool) {
	c.watchersmu.Lock()
	defer c.watchersmu.Unlock()

	newgvks := map[schema.GroupVersionKind]struct{}{}
	for _, status := range edgetask.Status.ResourcesStatus {
		gvk := schema.FromAPIVersionAndKind(status.APIVersion, status.Kind)
		newgvks[gvk] = struct{}{}
	}
	for gvk := range c.watchers {
		if _, ok := newgvks[gvk]; !ok {
			c.watchers[gvk].Stop()
			delete(c.watchers, gvk)
		} else {
			delete(newgvks, gvk)
		}
	}
	for gvk := range newgvks {
		go c.watch(gvk, callback)
	}
}

// nolint: funlen
func (c *perClientWatcher) watch(gvk schema.GroupVersionKind, callback func(obj client.Object) bool) {
	log := logr.FromContextOrDiscard(c.basectx).WithValues("gvk", gvk)
	log.Info("start list-watch")
	defer log.Info("stop list-watch")

	c.watchersmu.Lock()
	_, ok := c.watchers[gvk]
	if ok {
		c.watchersmu.Unlock()
		return
	}

	initlist := kube.NewListOf(gvk, c.cli.Scheme()) // use typed list if possible
	// list all objects
	if err := c.cli.List(c.basectx, initlist); err != nil {
		log.Error(err, "failed to list objects")
		c.watchersmu.Unlock()
		return
	}

	if err := apimeta.EachListItem(initlist, func(o runtime.Object) error {
		if obj, ok := o.(client.Object); ok {
			if callback(obj) {
				return errors.New("stop watching")
			}
		}
		return nil
	}); err != nil {
		return // stop watching
	}

	log.Info("start watching", "gvk", gvk)
	defer log.Info("stop watching", "gvk", gvk)

	watcher, err := c.cli.Watch(c.basectx, initlist)
	if err != nil {
		c.watchersmu.Unlock()
		return
	}
	defer watcher.Stop()

	c.watchers[gvk] = watcher // watch canbe closed from outside
	c.watchersmu.Unlock()

	defer delete(c.watchers, gvk)

	for {
		select {
		case <-c.basectx.Done():
			return
		case e, ok := <-watcher.ResultChan():
			if !ok {
				return
			}
			switch e.Type {
			case watch.Added, watch.Modified:
				if obj, ok := e.Object.(client.Object); ok {
					if callback(obj) {
						return
					}
				}
			case watch.Error:
				return
			}
		}
	}
}
