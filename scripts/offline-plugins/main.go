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
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/exp/slices"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	plugins "kubegems.io/kubegems/pkg/apis/plugins"
	pluginv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/bundle"
	"kubegems.io/kubegems/pkg/installer/bundle/helm"
	"kubegems.io/kubegems/pkg/installer/pluginmanager"
	"kubegems.io/kubegems/pkg/installer/utils"
)

var _ = pluginv1beta1.AddToScheme(scheme.Scheme)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()

	from := flag.String("from", "plugins.txt", "offline plugin versions from")
	to := flag.String("to", "bin/plugins", "offline plugins to")
	flag.Parse()

	if err := Run(ctx, *from, *to); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func Run(ctx context.Context, from, to string) error {
	cached, err := ReadFromFileOrLatest(ctx, from)
	if err != nil {
		return err
	}
	if err := Download(ctx, to, cached); err != nil {
		return err
	}
	return nil
}

func ReadFromFileOrLatest(ctx context.Context, filename string) ([]OfflinePlugin, error) {
	caches, err := ReadPluginFile(filename)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		// ignore not found
	}
	if len(caches) == 0 {
		pmr := &pluginmanager.Repository{Address: plugins.KubegemsChartsRepoURL}
		if err := pmr.RefreshRepoIndex(ctx); err != nil {
			return nil, err
		}
		latestCaches := readLatestPluginVersions(ctx, pmr)
		// writeback
		if err := WritePluginFile(filename, latestCaches); err != nil {
			return nil, err
		}
		caches = latestCaches
	}
	return caches, nil
}

func readCachedVersions(ctx context.Context, repo *pluginmanager.Repository, cached []OfflinePlugin) map[string]*pluginmanager.PluginVersion {
	ret := map[string]*pluginmanager.PluginVersion{}
	for _, cache := range cached {
		found := false
		for _, cv := range repo.Plugins[cache.Name] {
			if cv.Version == cache.Version {
				ret[cache.Name] = &cv
				found = true
				log.Printf("found: %s-%s", cache.Name, cache.Version)
				break
			}
		}
		if !found {
			log.Printf("version not found: %s-%s", cache.Name, cache.Version)
		}
	}
	return ret
}

func readLatestPluginVersions(ctx context.Context, repo *pluginmanager.Repository) []OfflinePlugin {
	ret := []OfflinePlugin{}
	for name, versions := range repo.Plugins {
		// do not download kubegems charts,it exists locally.
		if _, err := os.Stat(filepath.Join("deploy", "plugins", name)); err == nil {
			continue
		}
		// find latest version match kubegems
		if len(versions) == 0 {
			log.Printf("no version found for plugin %s", name)
			continue
		}
		latest := versions[0]
		log.Printf("found latest: %s-%s", name, latest.Version)
		ret = append(ret, OfflinePlugin{
			Repository: latest.Repository,
			Name:       latest.Name,
			Version:    latest.Version,
		})
	}
	return ret
}

func Download(ctx context.Context, basedir string, list []OfflinePlugin) error {
	applier := bundle.NewDefaultApply(nil, nil, &bundle.Options{CacheDir: basedir})
	for _, pv := range list {
		plugin := &pluginv1beta1.Plugin{
			ObjectMeta: v1.ObjectMeta{Name: pv.Name, Namespace: "default"},
			Spec: pluginv1beta1.PluginSpec{
				URL:     pv.Repository,
				Version: pv.Version,
				Kind:    pluginv1beta1.BundleKindTemplate,
				Path:    pv.Path,
			},
		}
		log.Printf("template %s-%s using %s", plugin.Name, plugin.Spec.Version, plugin.Spec.Kind)
		manifest, err := applier.Template(ctx, plugin)
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
	indexpath := bundle.PerRepoCacheDir(plugins.KubegemsChartsRepoURL, basedir)
	log.Printf("generating helm repo index.yaml under %s", indexpath)
	i, err := repo.IndexDirectory(indexpath, "")
	if err != nil {
		return err
	}
	i.SortEntries()
	return i.WriteFile(filepath.Join(indexpath, "index.yaml"), helm.DefaultFileMode)
}

func ReadPluginFile(filename string) ([]OfflinePlugin, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseContent(data), nil
}

func WritePluginFile(filename string, list []OfflinePlugin) error {
	data := bytes.NewBuffer(nil)
	slices.SortFunc(list, func(a, b OfflinePlugin) int {
		return strings.Compare(a.String(), b.String())
	})
	for _, val := range list {
		fmt.Fprintf(data, "%s %s %s\n", val.Repository, val.Name, val.Version)
	}
	return os.WriteFile(filename, data.Bytes(), helm.DefaultFileMode)
}

type OfflinePlugin struct {
	Repository string
	Name       string
	Version    string
	Path       string
}

func (o *OfflinePlugin) String() string {
	return fmt.Sprintf("%s/%s@%s", o.Repository, o.Name, o.Version)
}

func ParseContent(data []byte) []OfflinePlugin {
	scan := bufio.NewReader(bytes.NewReader(data))
	ret := []OfflinePlugin{}
	for {
		line, _, err := scan.ReadLine()
		if err == io.EOF {
			break
		}
		line = bytes.TrimLeft(line, " ")
		if len(line) == 0 {
			continue
		}
		if line[0] == '#' || line[0] == ';' {
			continue
		}
		fields := bytes.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch len(fields) {
		case 0, 1:
			continue
		case 2:
			ret = append(ret, OfflinePlugin{
				Repository: string(fields[0]),
				Name:       string(fields[0]),
			})
		case 3:
			ret = append(ret, OfflinePlugin{
				Repository: string(fields[0]),
				Name:       string(fields[1]),
				Version:    string(fields[2]),
			})
		default:
			ret = append(ret, OfflinePlugin{
				Repository: string(fields[0]),
				Name:       string(fields[1]),
				Version:    string(fields[2]),
				Path:       string(fields[3]),
			})
		}
	}
	return ret
}
