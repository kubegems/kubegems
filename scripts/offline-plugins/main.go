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
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/kubernetes/scheme"
	plugins "kubegems.io/kubegems/pkg/apis/plugins"
	pluginv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/bundle"
	"kubegems.io/kubegems/pkg/installer/bundle/helm"
	"kubegems.io/kubegems/pkg/installer/pluginmanager"
	"kubegems.io/kubegems/pkg/installer/utils"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()

	// download latest charts
	if err := DownloadLatestCharts(ctx, plugins.KubegemsChartsRepoURL, "bin/plugins", "latest"); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

var _ = pluginv1beta1.AddToScheme(scheme.Scheme)

func DownloadLatestCharts(ctx context.Context, repoaddr string, into string, kubegemsVersion string) error {
	applier := bundle.NewDefaultApply(nil, nil, &bundle.Options{CacheDir: into})
	pluginrepo := pluginmanager.Repository{Address: repoaddr}
	if err := pluginrepo.RefreshRepoIndex(ctx); err != nil {
		return err
	}
	kubegemsExample := pluginmanager.PluginVersion{Name: "kubegems", Version: kubegemsVersion}
	for name, versions := range pluginrepo.Plugins {
		// do not download kubegems charts,it exists locally.
		if strings.HasPrefix(name, "kubegems") {
			continue
		}
		var cacheVersion *pluginmanager.PluginVersion
		// find latest version match kubegems
		for _, item := range versions {
			if err := pluginmanager.CheckDependecy(item.Requirements, kubegemsExample); err != nil {
				log.Printf("ignore plugin [%s-%s] on dependencis not match: %s", item.Name, item.Version, err.Error())
				continue
			} else {
				cacheVersion = &item
				break
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
			if _, err := applier.Download(ctx, plugin); err != nil {
				log.Printf("on download: %v", err)
				return err
			}
		}
	}
	// build index
	indexpath := bundle.PerRepoCacheDir(repoaddr, into)
	log.Printf("generating helm repo index.yaml under %s", indexpath)
	i, err := repo.IndexDirectory(indexpath, "")
	if err != nil {
		return err
	}
	i.SortEntries()
	return i.WriteFile(filepath.Join(indexpath, "index.yaml"), helm.DefaultFileMode)
}
