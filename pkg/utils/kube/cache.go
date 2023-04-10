package kube

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var DefaultResyncTime = 10 * time.Hour

type CacheClient struct {
	client.Client
	// nolint: containedctx
	basectx      context.Context
	caches       map[schema.GroupVersionKind]client.Reader
	mu           sync.RWMutex
	eventhandler cache.ResourceEventHandler
}

func NewCachedClient(ctx context.Context, cli client.Client, eventhandler cache.ResourceEventHandler) *CacheClient {
	return &CacheClient{
		basectx:      ctx,
		Client:       cli,
		eventhandler: eventhandler,
		caches:       make(map[schema.GroupVersionKind]client.Reader),
	}
}

func (m *CacheClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	reader, err := m.getReader(obj)
	if err != nil {
		return err
	}
	return reader.Get(ctx, key, obj)
}

func (m *CacheClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	reader, err := m.getReader(list)
	if err != nil {
		return err
	}
	return reader.List(ctx, list, opts...)
}

func (m *CacheClient) getReader(example runtime.Object) (client.Reader, error) {
	gvk, err := apiutil.GVKForObject(example, m.Client.Scheme())
	if err != nil {
		return nil, err
	}
	if apimeta.IsListType(example) {
		gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	}
	m.mu.RLock()
	reader, ok := m.caches[gvk]
	m.mu.RUnlock()
	if ok {
		return reader, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	reader, err = NewCacheReaderFor(m.basectx, m.Client, gvk, m.eventhandler)
	if err != nil {
		return nil, err
	}
	m.caches[gvk] = reader
	return reader, nil
}

func NewCacheReaderFor(ctx context.Context, cli client.Client,
	gvk schema.GroupVersionKind, eventhandler cache.ResourceEventHandler,
	listoptions ...client.ListOption,
) (client.Reader, error) {
	example, err := cli.Scheme().New(gvk)
	if err != nil {
		return nil, err
	}
	example.GetObjectKind().SetGroupVersionKind(gvk)

	listgvk := gvk.GroupVersion().WithKind(gvk.Kind + "List")
	listObj, err := cli.Scheme().New(listgvk)
	if err != nil {
		return nil, err
	}
	listObj.GetObjectKind().SetGroupVersionKind(listgvk)

	objlist, ok := listObj.(client.ObjectList)
	if !ok {
		return nil, fmt.Errorf("not a client.ObjectList: %T", listObj)
	}
	watchcli, ok := cli.(client.WithWatch)
	if !ok {
		return nil, fmt.Errorf("not a client.WithWatch: %T", cli)
	}
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			//nolint: forcetypeassert
			list := objlist.DeepCopyObject().(client.ObjectList)
			err := cli.List(ctx, list, append(listoptions, &client.ListOptions{Raw: &options})...)
			return list, err
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return watchcli.Watch(ctx, objlist, append(listoptions, &client.ListOptions{Raw: &options})...)
		},
	}
	informer := cache.NewSharedIndexInformer(lw, example, DefaultResyncTime, defaultIndexersOf(gvk))
	if eventhandler != nil {
		informer.AddEventHandler(eventhandler)
	}
	go informer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		return nil, fmt.Errorf("cache sync failed")
	}
	cacheReader := NewCacheReader(informer.GetIndexer(), gvk, meta.RESTScopeNameNamespace, false)
	return cacheReader, nil
}

func defaultIndexersOf(gvk schema.GroupVersionKind) cache.Indexers {
	ret := cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	}
	if corev1.SchemeGroupVersion.WithKind("Event") == gvk {
		ret["field:involvedObject.uid"] = eventUIDIndexer
	}
	return ret
}

// "Index with name field:involvedObject.uid does not exist"
func eventUIDIndexer(obj any) ([]string, error) {
	event, ok := obj.(*corev1.Event)
	if !ok {
		return nil, fmt.Errorf("not a corev1.Event: %T", obj)
	}
	return []string{
		event.Namespace + "/" + string(event.InvolvedObject.UID),
	}, nil
}
