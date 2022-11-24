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

package cluster

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsvebeta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"kubegems.io/kubegems/pkg/agent/indexer"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

type Interface interface {
	cluster.Cluster
	Config() *rest.Config
	Kubernetes() kubernetes.Interface
	Discovery() discovery.CachedDiscoveryInterface
	Watch(ctx context.Context, list client.ObjectList, callback func(watch.Event) error, opts ...client.ListOption) error
}

type Cluster struct {
	cluster.Cluster
	config     *rest.Config
	discovery  discovery.CachedDiscoveryInterface
	kubernetes kubernetes.Interface
}

func WithDisableCaches() func(o *cluster.Options) {
	disabled := []client.Object{
		&metricsvebeta1.NodeMetrics{},
		&metricsvebeta1.PodMetrics{},
	}
	return func(o *cluster.Options) { o.ClientDisableCacheFor = append(o.ClientDisableCacheFor, disabled...) }
}

type WatchableDelegatingClient struct {
	client.Client
	watchable client.WithWatch
}

func (c *WatchableDelegatingClient) Watch(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	return c.watchable.Watch(ctx, obj, opts...)
}

func WithWatchClient(o *cluster.Options) {
	o.NewClient = func(cache cache.Cache, config *rest.Config, options client.Options, uncachedObjects ...client.Object) (client.Client, error) {
		c, err := client.NewWithWatch(config, options)
		if err != nil {
			return nil, err
		}
		delegating, err := client.NewDelegatingClient(client.NewDelegatingClientInput{
			CacheReader:       cache,
			Client:            c,
			UncachedObjects:   uncachedObjects,
			CacheUnstructured: true,
		})
		if err != nil {
			return nil, err
		}
		return &WatchableDelegatingClient{watchable: c, Client: delegating}, nil
	}
}

func WithDefaultScheme(o *cluster.Options) {
	o.Scheme = kube.GetScheme()
}

func NewClusterAndStart(ctx context.Context, config *rest.Config, options ...cluster.Option) (*Cluster, error) {
	c, err := NewCluster(config, options...)
	if err != nil {
		return nil, err
	}
	if err := indexer.CustomIndexPods(c.GetCache()); err != nil {
		return nil, err
	}
	go c.Start(ctx)
	c.GetCache().WaitForCacheSync(ctx)
	return c, nil
}

func NewCluster(config *rest.Config, options ...cluster.Option) (*Cluster, error) {
	discovery, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	options = append(options,
		WithDefaultScheme,
		WithDisableCaches(),
		WithWatchClient)

	kubernetesClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	c, err := cluster.New(config, options...)
	if err != nil {
		return nil, err
	}
	return &Cluster{
		Cluster:    c,
		kubernetes: kubernetesClientSet,
		config:     config,
		discovery:  memory.NewMemCacheClient(discovery),
	}, nil
}

func (c *Cluster) Kubernetes() kubernetes.Interface {
	return c.kubernetes
}

func (c *Cluster) Config() *rest.Config {
	return c.config
}

func (c *Cluster) Discovery() discovery.CachedDiscoveryInterface {
	return c.discovery
}

func (c *Cluster) Watch(ctx context.Context, list client.ObjectList, callback func(watch.Event) error, opts ...client.ListOption) error {
	gvk, err := apiutil.GVKForObject(list, c.GetScheme())
	if err != nil {
		return err
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")

	mapping, err := c.Cluster.GetRESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	if callback == nil {
		return errors.NewBadRequest("no callback provided")
	}

	listOpts := client.ListOptions{}
	listOpts.ApplyOptions(opts)

	config := c.config
	nclient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	watcher, err := nclient.
		Resource(mapping.Resource).
		Namespace(listOpts.Namespace).
		Watch(ctx, *listOpts.AsListOptions())
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case event := <-watcher.ResultChan():
			if err := callback(event); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}
