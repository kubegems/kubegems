package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"kubegems.io/pkg/installer/controllers/gitops"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	ManagedPluginAnnotation = "kubegems.io/plugin-name"
)

type NativePlugin struct {
	Config       *rest.Config
	CacheDir     string
	BuildFunc    BuildFunc
	gitopsengine *gitops.GitOpsEngine
}

func WithManagedResourceSelectByPluginName(namespace, name string) gitops.Option {
	const CountNameAndNamespace = 2
	return gitops.WithManagedResourceSelection(func(obj client.Object) bool {
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

type BuildFunc func(ctx context.Context, plugin Plugin) ([]*unstructured.Unstructured, error)

func NewNativePlugin(restconfig *rest.Config, defaultrepo string, buildfun BuildFunc) *NativePlugin {
	if abs, _ := filepath.Abs(defaultrepo); abs != defaultrepo {
		defaultrepo = abs
	}
	return &NativePlugin{Config: restconfig, CacheDir: defaultrepo, BuildFunc: buildfun}
}

func (n *NativePlugin) build(ctx context.Context, plugin Plugin) ([]*unstructured.Unstructured, error) {
	if err := DownloadPlugin(ctx, &plugin, n.CacheDir); err != nil {
		return nil, err
	}
	manifests, err := n.BuildFunc(ctx, plugin)
	if err != nil {
		return nil, fmt.Errorf("build manifests: %v", err)
	}
	// add inline resources
	inlineresources, err := InlineBuildPlugin(ctx, plugin)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, inlineresources...)
	for i := range manifests {
		annotations := manifests[i].GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[ManagedPluginAnnotation] = fmt.Sprintf("%s/%s", plugin.Namespace, plugin.Name)
		manifests[i].SetAnnotations(annotations)
	}
	return manifests, nil
}

func (n *NativePlugin) Apply(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("name", plugin.Name, "namespace", plugin.Namespace)

	// check already uptodate
	if status.Phase == pluginsv1beta1.PluginPhaseInstalled && reflect.DeepEqual(status.Values, plugin.Values) {
		log.Info("plugin is uptodate and no changes")
		return nil
	}

	// build manifests
	manifests, err := n.build(ctx, plugin)
	if err != nil {
		return err
	}
	status.Resources = manifests

	// apply
	var result *gitops.SyncResult
	if plugin.DryRun {
		result = &gitops.SyncResult{Phase: common.OperationSucceeded, Message: "dry run succeeded"}
	} else {
		result, err = n.apply(ctx, plugin.Namespace, manifests,
			WithManagedResourceSelectByPluginName(plugin.Namespace, plugin.Name))
		if err != nil {
			return err
		}
	}

	if err := n.parseResult(result, status); err != nil {
		return err
	}
	now := metav1.Now()
	// installed
	status.Phase = pluginsv1beta1.PluginPhaseInstalled
	status.Values = plugin.Values
	status.Message = result.Message
	status.Name, status.Namespace = plugin.Name, plugin.Namespace
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

	var result *gitops.SyncResult
	var err error

	if plugin.DryRun {
		result = &gitops.SyncResult{Phase: common.OperationSucceeded, Message: "dry run succeeded"}
	} else {
		result, err = n.apply(ctx, namespace, []*unstructured.Unstructured{},
			WithManagedResourceSelectByPluginName(namespace, name))
		if err != nil {
			return err
		}
	}

	if err := n.parseResult(result, status); err != nil {
		return err
	}
	status.Phase = pluginsv1beta1.PluginPhaseRemoved
	status.Message = result.Message
	status.Name = plugin.Name
	status.Namespace = plugin.Namespace
	status.DeletionTimestamp = metav1.Now()
	return nil
}

func (n *NativePlugin) apply(ctx context.Context, namespace string, resources []*unstructured.Unstructured, options ...gitops.Option) (*gitops.SyncResult, error) {
	if n.gitopsengine == nil {
		n.gitopsengine = &gitops.GitOpsEngine{Config: n.Config}
	}
	return n.gitopsengine.Apply(ctx, namespace, resources, options...)
}

func (n *NativePlugin) parseResult(result *gitops.SyncResult, status *PluginStatus) error {
	if result.Phase == common.OperationRunning {
		return fmt.Errorf("sync is still running: %s", result.Message)
	}

	errmsgs := []string{}
	notes := []map[string]interface{}{}
	for _, result := range result.Results {
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
	return nil
}
