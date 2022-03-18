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
	"github.com/argoproj/gitops-engine/pkg/engine"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/argoproj/gitops-engine/pkg/sync"
	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/argoproj/gitops-engine/pkg/utils/tracing"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"
)

const (
	operationRefreshTimeout = time.Second * 5
	ManagedPluginAnnotation = "kubegems.io/plugin-name"
)

type NativeApplier struct {
	Client client.Client
	config *rest.Config

	cache       cache.ClusterCache
	engine      engine.GitOpsEngine
	manifestDir string
}

func NewNativeApplier(ctx context.Context, mgr manager.Manager, manifestDir string) (*NativeApplier, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("applier", "native")

	config := mgr.GetConfig()
	n := &NativeApplier{
		manifestDir: manifestDir,
		Client:      mgr.GetClient(),
		config:      config,
	}
	n.cache = cache.NewClusterCache(config,
		cache.SetLogr(log),
		cache.SetClusterResources(true),
		cache.SetPopulateResourceInfoHandler(n.parseResourceInfo),
	)
	if err := n.cache.EnsureSynced(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *NativeApplier) parseResourceInfo(un *unstructured.Unstructured, isRoot bool) (info interface{}, cacheManifest bool) {
	return nil, true
}

func (n *NativeApplier) isManagedByPlugin(name string) func(r *cache.Resource) bool {
	return func(r *cache.Resource) bool {
		if resource := r.Resource; resource != nil {
			if annotations := resource.GetAnnotations(); annotations != nil && annotations[ManagedPluginAnnotation] == name {
				return true
			}
		}
		return false
	}
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

func (n *NativeApplier) apply(ctx context.Context, resources []*unstructured.Unstructured, name, namespace string) (*syncResult, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("name", name, "namespace", namespace)

	managedResources, err := n.cache.GetManagedLiveObjs(resources, n.isManagedByPlugin(name))
	if err != nil {
		return nil, err
	}
	reconciliationResult := sync.Reconcile(resources, managedResources, namespace, n.cache)

	// diffRes, err := diff.DiffArray(reconciliationResult.Target, reconciliationResult.Live, diff.WithLogr(log))
	// if err != nil {
	// 	return nil, err
	// }

	opts := []sync.SyncOpt{
		sync.WithSkipHooks(true),
		sync.WithLogr(log),
		sync.WithPrune(true),
		sync.WithHealthOverride(alwaysHealthOverride{}),
		sync.WithNamespaceCreation(true, func(u *unstructured.Unstructured) bool { return true }),
	}

	kubectl := &kube.KubectlCmd{Log: log, Tracer: tracing.NopTracer{}}
	syncCtx, cleanup, err := sync.NewSyncContext("", reconciliationResult, n.config, n.config, kubectl, namespace, n.cache.GetOpenAPISchema(), opts...)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	defer syncCtx.Terminate()

	var result *syncResult
	for i := 0; i < 3; i++ {
		syncCtx.Sync()
		phase, message, resources := syncCtx.GetState()
		result = &syncResult{phase: phase, message: message, results: resources}
		if phase.Completed() {
			if phase == common.OperationError {
				err = fmt.Errorf("sync operation failed: %s", message)
			}
			return result, err
		}
		time.Sleep(operationRefreshTimeout)
	}
	return result, nil
}

// nolint: funlen
func (n *NativeApplier) Apply(
	ctx context.Context,
	plugin pluginsv1beta1.InstallerSpecPlugin,
	globalValues map[string]interface{},
	status *pluginsv1beta1.InstallerStatusStatus,
) error {
	namespace, name := plugin.Namespace, plugin.Name
	log := logr.FromContextOrDiscard(ctx).WithValues("name", name, "namespace", namespace)

	enabled := plugin.Enabled
	var err error

	var manifests []*unstructured.Unstructured
	if enabled {
		manifests, _, err = n.parseManifests(plugin, globalValues)
		if err != nil {
			return err
		}
	}

	if (!enabled && status.Status == pluginsv1beta1.StatusUninstalled) ||
		(enabled && status.Status == pluginsv1beta1.StatusDeployed) && reflect.DeepEqual(status.Values, plugin.Values) {
		log.Info("plugin is uptodate and no changes")
		return nil
	}

	state, err := n.apply(ctx, manifests, name, namespace)
	if err != nil {
		return err
	}

	switch state.phase {
	case common.OperationRunning:
		return fmt.Errorf("sync is still running: %s", state.message)
	}

	errmsgs := []string{}
	notes := []map[string]interface{}{}
	for _, result := range state.results {
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
	if !enabled {
		// uninstalled
		status.Status = pluginsv1beta1.StatusUninstalled
		if status.CreationTimestamp.IsZero() {
			status.CreationTimestamp = metav1.Now()
		}
		status.UpgradeTimestamp = metav1.Now()
	} else {
		// installed
		status.Status = pluginsv1beta1.StatusDeployed
		status.DeletionTimestamp = &now
	}

	status.Values = plugin.Values
	status.Message = state.message

	return nil
}

type TemplatesValues struct {
	Values  map[string]interface{}
	Release map[string]interface{}
}

func (n *NativeApplier) parseManifests(
	plugin pluginsv1beta1.InstallerSpecPlugin,
	globalValues map[string]interface{},
) ([]*unstructured.Unstructured, string, error) {
	// tmplate vals
	name, namespace := plugin.Name, plugin.Namespace
	tplValues := TemplatesValues{
		Values: func() map[string]interface{} {
			vals := UnmarshalValues(plugin.Values)
			if vals == nil {
				vals = map[string]interface{}{}
			}
			vals["global"] = globalValues
			return vals
		}(),
		Release: map[string]interface{}{
			"Name":      name,
			"Namespace": namespace,
		},
	}
	var res []*unstructured.Unstructured
	if err := filepath.Walk(filepath.Join(n.manifestDir, name), func(path string, info os.FileInfo, err error) error {
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
		data, err = templates(info.Name(), data, tplValues)
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
		return nil, "", err
	}
	for i := range res {
		annotations := res[i].GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[ManagedPluginAnnotation] = name
		res[i].SetAnnotations(annotations)

		// we remove namespace to avoid cross namespace conflicts
		res[i].SetNamespace("")
	}
	return res, "", nil
}

func templates(name string, content []byte, values interface{}) ([]byte, error) {
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
