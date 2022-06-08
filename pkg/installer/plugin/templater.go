package plugin

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
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"sigs.k8s.io/yaml"
)

type Templater struct {
	Config *rest.Config
}

// TemplatesTemplate using helm template engine to render,but allow apply to different namespaces
func (t Templater) Template(ctx context.Context, plugin *pluginsv1beta1.Plugin, dir string) ([]byte, error) {
	chart, err := Load(plugin.Name, plugin.Spec.Version, dir)
	if err != nil {
		return nil, err
	}
	plugin.Status.Version = chart.Metadata.Version

	options := chartutil.ReleaseOptions{
		Name:      plugin.Name,
		Namespace: plugin.Namespace,
		IsInstall: true,
	}

	caps := chartutil.DefaultCapabilities
	valuesToRender, err := chartutil.ToRenderValues(chart, plugin.Spec.Values.Object, options, caps)
	if err != nil {
		return nil, err
	}

	if vals, ok := valuesToRender.AsMap()["Values"].(chartutil.Values); ok {
		plugin.Spec.Values = pluginsv1beta1.Values{Object: vals}
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

func Load(name, version, path string) (*chart.Chart, error) {
	if version == "" {
		version = "0.0.0"
	}
	absdir, err := filepath.Abs(path)
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
			Name:       name,
			Version:    version,
		}
		chartfilecontent, err := yaml.Marshal(chartfile)
		if err != nil {
			return nil, err
		}
		files = append(files, &loader.BufferedFile{Name: chartFileName, Data: chartfilecontent})
	}
	return loader.LoadFiles(files)
}
