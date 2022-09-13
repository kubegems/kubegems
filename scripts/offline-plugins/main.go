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

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/kubernetes/scheme"
	pluginv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/bundle"
	"kubegems.io/kubegems/pkg/installer/bundle/helm"
	"kubegems.io/kubegems/pkg/installer/utils"
	"kubegems.io/kubegems/pkg/utils/gemsplugin"
)

func main() {
	kubegemsrepo := "https://charts.kubegems.io/kubegems"
	ctx := context.Background()
	// download latest charts
	if err := DownloadLatestCharts(ctx, kubegemsrepo, "bin/plugins", "latest"); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

var _ = pluginv1beta1.AddToScheme(scheme.Scheme)

func DownloadLatestCharts(ctx context.Context, repoaddr string, into string, kubegemsVersion string) error {
	repourl, err := url.Parse(repoaddr)
	if err != nil {
		return err
	}

	applier := bundle.NewDefaultApply(nil, nil, &bundle.Options{CacheDir: into})

	pluginrepo := gemsplugin.Repository{Address: repoaddr}
	if err := pluginrepo.RefreshRepoIndex(ctx); err != nil {
		return err
	}

	kubegemsExample := gemsplugin.PluginVersion{Name: "kubegems", Version: kubegemsVersion}

	for name, versions := range pluginrepo.Plugins {
		var cacheVersion *gemsplugin.PluginVersion
		if strings.HasPrefix(name, "kubegems") {
			// cache kubegems version
			for _, item := range versions {
				if item.Version != kubegemsVersion {
					continue
				} else {
					cacheVersion = &item
					break
				}
			}
			if cacheVersion == nil {
				cacheVersion = &versions[0]
				// cacheVersion.Kind = pluginv1beta1.BundleKindHelm
				log.Printf("kubegems plugin %s version %s not found,use %s instead", name, kubegemsVersion, cacheVersion.Version)
			}
		} else {
			// find latest version match kubegems
			for _, item := range versions {
				if err := gemsplugin.CheckDependecy(item.Requirements, kubegemsExample); err != nil {
					continue
				} else {
					cacheVersion = &item
					break
				}
			}
		}
		if cacheVersion == nil {
			log.Printf("no matched version to cache on plugin %s", name)
			continue
		}
		log.Printf("download %s-%s from %s", cacheVersion.Name, cacheVersion.Version, repoaddr)
		manifest, err := applier.Template(ctx, cacheVersion.ToPlugin())
		if err != nil {
			log.Printf("on template: %v", err)
			return err
		}
		// parse plugins
		objs, err := utils.SplitYAMLFilterd[*pluginv1beta1.Plugin](bytes.NewReader(manifest))
		if err != nil {
			return err
		}
		// download plugin
		for _, plugin := range objs {
			name := plugin.Spec.Chart
			if name == "" {
				name = plugin.Name
			}
			log.Printf("download %s-%s from %s", name, plugin.Spec.Version, plugin.Spec.URL)
			if _, err := applier.Template(ctx, plugin); err != nil {
				log.Printf("on template: %v", err)
				return err
			}
		}
	}
	// build index
	// plugin cache helms in {hostname}/{charts}
	indexpath := filepath.Join(into, repourl.Host)
	log.Printf("generating helm repo index.yaml under %s", indexpath)
	i, err := repo.IndexDirectory(indexpath, "")
	if err != nil {
		return err
	}
	i.SortEntries()
	return i.WriteFile(filepath.Join(indexpath, "index.yaml"), helm.DefaultFileMode)
}
