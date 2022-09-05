// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package template

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/releaseutil"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"sigs.k8s.io/yaml"
)

func NewTemplaterFunc(cfg *rest.Config) func(ctx context.Context, bundle *pluginsv1beta1.Plugin, dir string) ([]byte, error) {
	tmplater := &Templater{Config: cfg}
	return tmplater.Template
}

type Templater struct {
	Config *rest.Config
	DC     discovery.CachedDiscoveryInterface
}

// TemplatesTemplate using helm template engine to render,but allow apply to different namespaces
func (t Templater) Template(ctx context.Context, plugin *pluginsv1beta1.Plugin, dir string) ([]byte, error) {
	chart, err := Load(plugin.Name, plugin.Spec.Version, dir)
	if err != nil {
		return nil, err
	}
	// update bundle status
	plugin.Spec.Version = chart.Metadata.Version

	options := chartutil.ReleaseOptions{
		Name: plugin.Name,
		Namespace: func() string {
			if ns := plugin.Spec.InstallNamespace; ns != "" {
				return ns
			}
			return plugin.Namespace
		}(),
		IsInstall: true,
	}

	caps := chartutil.DefaultCapabilities

	if t.Config != nil && t.DC == nil {
		cs, err := kubernetes.NewForConfig(t.Config)
		if err != nil {
			return nil, err
		}
		t.DC = memory.NewMemCacheClient(cs)
	}

	if dc := t.DC; dc != nil {
		kubeVersion, err := dc.ServerVersion()
		if err != nil {
			return nil, err
		}
		apiVersions, err := action.GetVersionSet(dc)
		if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
			return nil, fmt.Errorf("could not get apiVersions from Kubernetes: %w", err)
		}
		caps.APIVersions = apiVersions
		caps.KubeVersion = chartutil.KubeVersion{
			Version: kubeVersion.GitVersion,
			Major:   kubeVersion.Major,
			Minor:   kubeVersion.Minor,
		}
	}

	valuesToRender, err := chartutil.ToRenderValues(chart, plugin.Spec.Values.Object, options, caps)
	if err != nil {
		return nil, err
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
