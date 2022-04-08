package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/yaml"
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

func HelmBuildPlugin(ctx context.Context, plugin Plugin) ([]*unstructured.Unstructured, error) {
	return TemplatesBuildPlugin(ctx, plugin)
}

func InlineBuildPlugin(ctx context.Context, plugin Plugin) ([]*unstructured.Unstructured, error) {
	rss := make([]*unstructured.Unstructured, 0, len(plugin.Resources))
	for i, obj := range plugin.Resources {
		uns := &unstructured.Unstructured{}
		if obj.Object != nil {
			// already unmarshaled
			scheme.Scheme.Convert(obj.Object, uns, nil)
		} else {
			if err := json.Unmarshal(obj.Raw, uns); err != nil {
				return nil, fmt.Errorf("unmarshal resource[%d]: %v", i, err)
			}
		}
		rss = append(rss, uns)
	}
	return rss, nil
}

// TemplatesBuildPlugin using helm template engine to render,but allow apply to different namespaces
func TemplatesBuildPlugin(ctx context.Context, plugin Plugin) ([]*unstructured.Unstructured, error) {
	options := chartutil.ReleaseOptions{
		Name:      plugin.Name,
		Namespace: plugin.Namespace,
		IsInstall: true,
	}
	chart, err := Load(plugin)
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

const chartFileName = "Chart.yaml"

func Load(plugin Plugin) (*chart.Chart, error) {
	absdir, err := filepath.Abs(plugin.Path)
	if err != nil {
		return nil, err
	}
	absdir += string(filepath.Separator)
	containsChartFile := false
	files := []*loader.BufferedFile{}
	walk := func(name string, fi os.FileInfo, err error) error {
		relfilename := strings.TrimPrefix(name, absdir)
		if relfilename == "" {
			return nil
		}
		if relfilename == chartFileName {
			containsChartFile = true
		}
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		files = append(files, &loader.BufferedFile{Name: relfilename, Data: data})
		return nil
	}
	if err = filepath.Walk(absdir, walk); err != nil {
		return nil, err
	}

	if !containsChartFile {
		chartfile := chart.Metadata{
			APIVersion: chart.APIVersionV2,
			Name:       plugin.Name,
			Version: func() string {
				if plugin.Version != "" {
					return plugin.Version
				}
				return "0.0.0"
			}(),
		}
		chartfilecontent, err := yaml.Marshal(chartfile)
		if err != nil {
			return nil, err
		}
		files = append(files, &loader.BufferedFile{Name: chartFileName, Data: chartfilecontent})
	}
	return loader.LoadFiles(files)
}
