package controllers

import (
	"context"
	"fmt"

	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
)

func KustomizeBuildPlugin(ctx context.Context, plugin Plugin) ([]*unstructured.Unstructured, error) {
	return KustomizeBuild(ctx, plugin.Path)
}

// build kustomization
func KustomizeBuild(ctx context.Context, dir string) ([]*unstructured.Unstructured, error) {
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	m, err := k.Run(filesys.MakeFsOnDisk(), dir)
	if err != nil {
		return nil, err
	}
	yml, err := m.AsYaml()
	if err != nil {
		return nil, err
	}

	res := []*unstructured.Unstructured{}
	items, err := kube.SplitYAML(yml)
	if err != nil {
		return nil, fmt.Errorf("parse content [%s]: %v", string(yml), err)
	}
	res = append(res, items...)

	return res, nil
}

// TemplatesBuildPlugin using helm template engine to render,but allow apply to different namespaces
func TemplatesBuildPlugin(ctx context.Context, plugin Plugin) ([]*unstructured.Unstructured, error) {
	options := chartutil.ReleaseOptions{
		Name:      plugin.Name,
		Namespace: plugin.Namespace,
		IsInstall: true,
	}
	chart, err := loader.Load(plugin.Path)
	if err != nil {
		return nil, err
	}
	valuesToRender, err := chartutil.ToRenderValues(chart, plugin.Values, options, chartutil.DefaultCapabilities)
	if err != nil {
		return nil, err
	}
	renderdFiles, err := engine.Render(chart, valuesToRender)
	if err != nil {
		return nil, err
	}
	var res []*unstructured.Unstructured
	for _, content := range renderdFiles {
		items, err := kube.SplitYAML([]byte(content))
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			res = append(res, item)
		}
	}
	return res, nil
}
