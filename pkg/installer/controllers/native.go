package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/argoproj/gitops-engine/pkg/sync"
	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/argoproj/gitops-engine/pkg/utils/tracing"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	ManagedPluginAnnotation = "kubegems.io/plugin-name"
)

type NativePlugin struct {
	Config      *rest.Config
	DefaultRepo string
	BuildFunc   BuildFunc
}

type Release struct {
	Name      string
	Namespace string
	Version   string
}

type BuildFunc func(ctx context.Context, path string, release Release, values map[string]interface{}) ([]*unstructured.Unstructured, error)

func NewNativePlugin(restconfig *rest.Config, defaultrepo string, buildfun BuildFunc) *NativePlugin {
	return &NativePlugin{Config: restconfig, DefaultRepo: defaultrepo, BuildFunc: buildfun}
}

// nolint: funlen
func (n *NativePlugin) Apply(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	namespace, name := plugin.Namespace, plugin.Name
	log := logr.FromContextOrDiscard(ctx).WithValues("name", name, "namespace", namespace)

	repo, path := plugin.Repo, plugin.Path
	if repo == "" {
		// use default local repo
		repo = "file://" + n.DefaultRepo
	}
	if path == "" {
		path = plugin.Name
	}

	p, err := Download(ctx, repo, plugin.Version, path)
	if err != nil {
		return err
	}
	path = p

	release := Release{
		Name:      plugin.Name,
		Namespace: plugin.Namespace,
		Version:   plugin.Version,
	}
	manifests, err := n.BuildFunc(ctx, path, release, plugin.Values)
	if err != nil {
		return fmt.Errorf("build manifests: %v", err)
	}
	for i := range manifests {
		annotations := manifests[i].GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[ManagedPluginAnnotation] = fmt.Sprintf("%s/%s", plugin.Namespace, plugin.Name)
		manifests[i].SetAnnotations(annotations)
		// we remove namespace to avoid cross namespace conflicts
		manifests[i].SetNamespace("")
	}
	if status.Phase == pluginsv1beta1.PluginPhaseInstalled && reflect.DeepEqual(status.Values, plugin.Values) {
		log.Info("plugin is uptodate and no changes")
		return nil
	}

	result, err := ApplyNative(ctx, n.Config, namespace, manifests, WithManagedResourceSelectByPluginName(namespace, name))
	if err != nil {
		return err
	}
	switch result.phase {
	case common.OperationRunning:
		return fmt.Errorf("sync is still running: %s", result.message)
	}
	errmsgs := []string{}
	notes := []map[string]interface{}{}
	for _, result := range result.results {
		switch result.Status {
		case common.ResultCodeSyncFailed:
			errmsgs = append(errmsgs, fmt.Sprintf("%s: %s", result.ResourceKey.String(), result.Message))
		}
		notes = append(notes, map[string]interface{}{
			"resource": result.ResourceKey.String(),
			"status":   result.Status,
		})
	}
	content, _ := yaml.Marshal(notes)
	status.Notes = string(content)

	if len(errmsgs) > 0 {
		return fmt.Errorf(strings.Join(errmsgs, "\n"))
	}

	now := metav1.Now()
	// installed
	status.Phase = pluginsv1beta1.PluginPhaseInstalled
	status.Values = plugin.Values
	status.Message = result.message
	status.Name = plugin.Name
	status.Namespace = plugin.Namespace
	if status.CreationTimestamp.IsZero() {
		status.CreationTimestamp = now
	}
	status.UpgradeTimestamp = now
	return nil
}

func (n *NativePlugin) Remove(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	log := logr.FromContextOrDiscard(ctx)
	namespace, name := plugin.Namespace, plugin.Name

	switch status.Phase {
	case pluginsv1beta1.PluginPhaseInstalled, pluginsv1beta1.PluginPhaseFailed:
		// continue processing
	case pluginsv1beta1.PluginPhaseNone:
		log.Info("plugin is removed or not installed")
		return nil
	case "":
		log.Info("plugin is not installed set to not installed")
		status.Phase = pluginsv1beta1.PluginPhaseNone
		status.CreationTimestamp = metav1.Now()
		return nil
	default:
		return nil
	}

	result, err := ApplyNative(ctx, n.Config, namespace, []*unstructured.Unstructured{}, WithManagedResourceSelectByPluginName(namespace, name))
	if err != nil {
		return err
	}
	errmsgs := []string{}
	notes := []map[string]interface{}{}
	for _, result := range result.results {
		switch result.Status {
		case common.ResultCodeSyncFailed:
			errmsgs = append(errmsgs, fmt.Sprintf("%s: %s", result.ResourceKey.String(), result.Message))
		}
		notes = append(notes, map[string]interface{}{
			"resource": result.ResourceKey.String(),
			"status":   result.Status,
		})
	}
	content, _ := yaml.Marshal(notes)
	status.Notes = string(content)

	if len(errmsgs) > 0 {
		return fmt.Errorf(strings.Join(errmsgs, "\n"))
	}

	status.Phase = pluginsv1beta1.PluginPhaseRemoved
	status.Message = result.message
	status.Name = plugin.Name
	status.Namespace = plugin.Namespace
	status.DeletionTimestamp = metav1.Now()
	return nil
}

type syncResult struct {
	phase   common.OperationPhase
	message string
	results []common.ResourceSyncResult
}

type alwaysHealthOverride struct{}

func (alwaysHealthOverride) GetResourceHealth(_ *unstructured.Unstructured) (*health.HealthStatus, error) {
	return &health.HealthStatus{Status: health.HealthStatusHealthy, Message: "always heathy"}, nil
}

type Options struct {
	ManagedResourceSelection func(obj client.Object) bool
}

type Option func(*Options)

func WithManagedResourceSelection(fun func(obj client.Object) bool) Option {
	return func(o *Options) {
		o.ManagedResourceSelection = fun
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

func ApplyNative(ctx context.Context, config *rest.Config, namespace string, resources []*unstructured.Unstructured, options ...Option) (*syncResult, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("namespace", namespace)

	opts := &Options{}
	for _, opt := range options {
		opt(opts)
	}

	clusterCache := cache.NewClusterCache(config,
		cache.SetLogr(log),
		cache.SetClusterResources(true),
		cache.SetPopulateResourceInfoHandler(
			func(un *unstructured.Unstructured, isRoot bool) (info interface{}, cacheManifest bool) {
				return nil, true
			},
		),
	)
	if err := clusterCache.EnsureSynced(); err != nil {
		return nil, err
	}
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
		sync.WithSkipHooks(true),
		sync.WithLogr(log),
		sync.WithPrune(true),
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
