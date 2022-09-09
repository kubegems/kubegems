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

	"k8s.io/client-go/kubernetes/scheme"
	pluginv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/bundle"
	"kubegems.io/kubegems/pkg/installer/utils"
	"kubegems.io/kubegems/pkg/utils/gemsplugin"
)

func main() {
	kubegemsrepo := "https://charts.kubegems.io/kubegems"
	ctx := context.Background()
	// download latest charts
	if err := DownloadLatestCharts(ctx, kubegemsrepo, "bin/plugins"); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

var _ = pluginv1beta1.AddToScheme(scheme.Scheme)

func DownloadLatestCharts(ctx context.Context, repo string, into string) error {
	applier := bundle.NewDefaultApply(nil, nil, &bundle.Options{CacheDir: into})

	pluginrepo := gemsplugin.Repository{Address: repo}
	if err := pluginrepo.RefreshRepoIndex(ctx); err != nil {
		return err
	}

	for _, versions := range pluginrepo.ListPluginVersions() {
		version := versions[0]
		log.Printf("download %s-%s from %s", version.Name, version.Version, repo)
		manifest, err := applier.Template(ctx, version.ToPlugin())
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
	return nil
}
