package gitops

import (
	"context"
	"strings"
	"time"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/argoproj/gitops-engine/pkg/sync"
	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/argoproj/gitops-engine/pkg/utils/tracing"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IgnoreOptions  = "plugins.kubegems.io/ignore-options"
	IgnoreOnUpdate = "OnUpdate"
	IgnoreOnDelete = "OnDelete"
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

type SyncResult struct {
	Phase   common.OperationPhase
	Message string
	Results []common.ResourceSyncResult
}

type alwaysHealthOverride struct{}

func (alwaysHealthOverride) GetResourceHealth(_ *unstructured.Unstructured) (*health.HealthStatus, error) {
	return &health.HealthStatus{Status: health.HealthStatusHealthy, Message: "always heathy"}, nil
}

func (n *GitOpsEngine) Apply(ctx context.Context, namespace string, resources []*unstructured.Unstructured, options ...Option) (*SyncResult, error) {
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
		sync.WithResourcesFilter(filterByIgnoreOptions),
	}
	kubectl := &kube.KubectlCmd{Log: log, Tracer: tracing.NopTracer{}}
	syncCtx, cleanup, err := sync.NewSyncContext("", reconciliationResult, config, config, kubectl, namespace, clusterCache.GetOpenAPISchema(), syncopts...)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	defer syncCtx.Terminate()
	var result *SyncResult
	period := time.Second
	err = wait.PollUntil(period, func() (done bool, err error) {
		syncCtx.Sync()
		phase, message, resources := syncCtx.GetState()
		result = &SyncResult{Phase: phase, Message: message, Results: resources}
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

func filterByIgnoreOptions(key kube.ResourceKey, target *unstructured.Unstructured, live *unstructured.Unstructured) bool {
	switch {
	// remove
	case target == nil:
	// create
	case target != nil && live == nil:
	// update
	case target != nil && live != nil:
		if annotations := live.GetAnnotations(); annotations != nil {
			// ignore different on update
			if ignore, ok := annotations[IgnoreOptions]; ok && strings.Contains(ignore, IgnoreOnUpdate) {
				return false
			}
		}
	}
	return true
}
