package controllers

import (
	"context"
	"strings"
	"time"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/sync"
	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/argoproj/gitops-engine/pkg/utils/tracing"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GitOpsEngine struct {
	cache  cache.ClusterCache
	Config *rest.Config
}

type GitOpsEngineOptions struct {
	DryRun                   bool
	ManagedResourceSelection func(obj client.Object) bool
}

type Option func(*GitOpsEngineOptions)

func WithManagedResourceSelection(fun func(obj client.Object) bool) Option {
	return func(o *GitOpsEngineOptions) {
		o.ManagedResourceSelection = fun
	}
}

func WithDryRun(b bool) Option {
	return func(o *GitOpsEngineOptions) {
		o.DryRun = b
	}
}

func WithManagedResourceSelectByPluginName(namespace, name string) Option {
	const CountNameAndNamespace = 2
	return WithManagedResourceSelection(func(obj client.Object) bool {
		if annotations := obj.GetAnnotations(); annotations != nil {
			nm := annotations[ManagedPluginAnnotation]
			splits := strings.SplitN(nm, "/", CountNameAndNamespace)
			if len(splits) >= 2 && splits[0] == namespace && splits[1] == name {
				return true
			}
		}
		return false
	})
}

func (n *GitOpsEngine) Apply(ctx context.Context, namespace string, resources []*unstructured.Unstructured, options ...Option) (*syncResult, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("namespace", namespace)
	opts := &GitOpsEngineOptions{}
	for _, opt := range options {
		opt(opts)
	}

	if n.cache == nil {
		newcache := cache.NewClusterCache(n.Config, cache.SetLogr(log), cache.SetClusterResources(true),
			cache.SetPopulateResourceInfoHandler(func(un *unstructured.Unstructured, isRoot bool) (info interface{}, cacheManifest bool) {
				return nil, true
			}),
		)
		if err := newcache.EnsureSynced(); err != nil {
			return nil, err
		}
		n.cache = newcache
	}
	clusterCache, config := n.cache, n.Config
	managedResources, err := clusterCache.GetManagedLiveObjs(resources, func(r *cache.Resource) bool {
		if opts.ManagedResourceSelection == nil {
			return false // default select nothing
		}
		return opts.ManagedResourceSelection(r.Resource)
	})
	if err != nil {
		return nil, err
	}
	reconciliationResult := sync.Reconcile(resources, managedResources, namespace, clusterCache)
	syncopts := []sync.SyncOpt{
		sync.WithLogr(log), sync.WithOperationSettings(opts.DryRun, true, true, true),
		sync.WithHealthOverride(alwaysHealthOverride{}),
		sync.WithNamespaceCreation(true, func(u *unstructured.Unstructured) bool { return true }),
	}
	kubectl := &kube.KubectlCmd{Log: log, Tracer: tracing.NopTracer{}}
	syncCtx, cleanup, err := sync.NewSyncContext("", reconciliationResult, config, config, kubectl, namespace, clusterCache.GetOpenAPISchema(), syncopts...)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	defer syncCtx.Terminate()
	var result *syncResult
	period := time.Second
	err = wait.PollUntil(period, func() (done bool, err error) {
		syncCtx.Sync()
		phase, message, resources := syncCtx.GetState()
		result = &syncResult{phase: phase, message: message, results: resources}
		if phase.Completed() {
			return true, err
		}
		return false, nil
	}, ctx.Done())
	if err != nil {
		return result, err
	}
	return result, nil
}
