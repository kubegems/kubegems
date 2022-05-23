package controllers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/releaseutil"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/yaml"
)

func KustomizeTemplatePlugin(ctx context.Context, plugin *Plugin) ([]byte, error) {
	return KustomizeBuild(ctx, plugin.Path)
}

// build kustomization
func KustomizeBuild(ctx context.Context, dir string) ([]byte, error) {
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	m, err := k.Run(filesys.MakeFsOnDisk(), dir)
	if err != nil {
		return nil, err
	}
	yml, err := m.AsYaml()
	if err != nil {
		return nil, err
	}
	return []byte(yml), nil
}

func InlineTemplatePlugin(ctx context.Context, plugin *Plugin) ([]byte, error) {
	out := bytes.NewBuffer(nil)
	for _, obj := range plugin.Resources {
		out.WriteString("---\n")
		_, err := out.Write(obj.Raw)
		if err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

type Templater struct {
	Config *rest.Config
}

// TemplatesTemplate using helm template engine to render,but allow apply to different namespaces
func (t Templater) Template(ctx context.Context, plugin *Plugin) ([]byte, error) {
	options := chartutil.ReleaseOptions{
		Name:      plugin.Name,
		Namespace: plugin.Namespace,
		IsInstall: true,
	}
	chart, err := Load(plugin)
	if err != nil {
		return nil, err
	}

	caps := chartutil.DefaultCapabilities
	valuesToRender, err := chartutil.ToRenderValues(chart, plugin.Values, options, caps)
	if err != nil {
		return nil, err
	}

	if vals, ok := valuesToRender.AsMap()["Values"].(chartutil.Values); ok {
		plugin.FullValues = vals
	}

	var renderdFiles map[string]string
	if t.Config != nil {
		renderdFiles, err = engine.RenderWithClient(chart, valuesToRender, t.Config)
		if err != nil {
			return nil, err
		}
	} else {
		renderdFiles, err = engine.Render(chart, valuesToRender)
		if err != nil {
			return nil, err
		}
	}
	_, manifests, err := releaseutil.SortManifests(renderdFiles, caps.APIVersions, releaseutil.InstallOrder)
	if err != nil {
		out := os.Stderr
		for file, val := range renderdFiles {
			fmt.Fprintf(out, "---\n# Source: %s\n%s\n", file, val)
		}
		fmt.Fprintln(out, "---")
		return nil, err
	}
	out := bytes.NewBuffer(nil)
	for _, crd := range chart.CRDObjects() {
		fmt.Fprintf(out, "---\n# Source: %s\n%s\n", crd.Name, string(crd.File.Data[:]))
	}
	for _, m := range manifests {
		fmt.Fprintf(out, "---\n# Source: %s\n%s\n", m.Name, m.Content)
	}
	return out.Bytes(), nil
}

const chartFileName = "Chart.yaml"

func Load(plugin *Plugin) (*chart.Chart, error) {
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
