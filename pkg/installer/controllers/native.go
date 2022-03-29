package controllers

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
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

type NativeApplier struct {
	config      *rest.Config
	manifestDir string
}

func NewNativeApplier(restconfig *rest.Config, manifestDir string) *NativeApplier {
	return &NativeApplier{manifestDir: manifestDir, config: restconfig}
}

// nolint: funlen
func (n *NativeApplier) Apply(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	namespace, name := plugin.Namespace, plugin.Name
	log := logr.FromContextOrDiscard(ctx).WithValues("name", name, "namespace", namespace)

	repo, path := plugin.Repo, plugin.Path
	if repo == "" {
		// use default local repo
		repo = "file://" + n.manifestDir
	}
	if path == "" {
		path = plugin.Name
	}

	p, err := Download(ctx, repo, plugin.Version, path)
	if err != nil {
		return err
	}
	path = p

	// parse manifests
	tplValues := TemplatesValues{
		Values:  plugin.Values,
		Release: map[string]interface{}{"Name": name, "Namespace": namespace},
	}
	manifests, err := ParseManifests(path, tplValues)
	if err != nil {
		return fmt.Errorf("parse manifests: %w", err)
	}
	for i := range manifests {
		annotations := manifests[i].GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[ManagedPluginAnnotation] = name
		manifests[i].SetAnnotations(annotations)
		// we remove namespace to avoid cross namespace conflicts
		manifests[i].SetNamespace("")
	}
	if status.Phase == pluginsv1beta1.PluginPhaseInstalled && reflect.DeepEqual(status.Values, plugin.Values) {
		log.Info("plugin is uptodate and no changes")
		return nil
	}

	result, err := ApplyNative(ctx, n.config, namespace, manifests, WithManagedResourceSelectByPluginName(name))
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

func (n *NativeApplier) Remove(ctx context.Context, plugin Plugin, status *PluginStatus) error {
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

	result, err := ApplyNative(ctx, n.config, namespace, []*unstructured.Unstructured{}, WithManagedResourceSelectByPluginName(name))
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

type TemplatesValues struct {
	Values  map[string]interface{}
	Release map[string]interface{}
}

func ParseManifests(path string, values interface{}) ([]*unstructured.Unstructured, error) {
	var res []*unstructured.Unstructured
	if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if ext := strings.ToLower(filepath.Ext(info.Name())); ext != ".json" && ext != ".yml" && ext != ".yaml" {
			return nil
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		// template
		data, err = Templates(info.Name(), data, values)
		if err != nil {
			return err
		}
		items, err := kube.SplitYAML(data)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %v", path, err)
		}
		res = append(res, items...)
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

func Templates(name string, content []byte, values interface{}) ([]byte, error) {
	template, err := template.
		New(name).
		Option("missingkey=zero").
		Funcs(sprig.TxtFuncMap()).
		Parse(string(content))
	if err != nil {
		return nil, err
	}
	result := bytes.NewBuffer(nil)
	if err := template.Execute(result, values); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
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

func WithManagedResourceSelectByPluginName(name string) Option {
	return WithManagedResourceSelection(func(obj client.Object) bool {
		if annotations := obj.GetAnnotations(); annotations != nil && annotations[ManagedPluginAnnotation] == name {
			return true
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
