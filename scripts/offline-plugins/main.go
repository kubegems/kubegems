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
	offline := NewOffline(plugins.KubegemsChartsRepoURL)
	if err := offline.ReadFromFile(ctx, from); err != nil {
		return err
	}
	if err := offline.Download(ctx, to); err != nil {
		return err
	}
	return nil
}

func NewOffline(repoaddr string) *Offline {
	return &Offline{
		repo:           pluginmanager.Repository{Address: repoaddr},
		offlineplugins: map[string]*pluginmanager.PluginVersion{},
	}
}

type Offline struct {
	repo           pluginmanager.Repository
	offlineplugins map[string]*pluginmanager.PluginVersion
}

func (o *Offline) ReadFromFile(ctx context.Context, filename string) error {
	caches, err := ReadPluginFile(filename)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		// ignore not found
	}
	if err := o.repo.RefreshRepoIndex(ctx); err != nil {
		return err
	}
	if len(caches) == 0 {
		o.offlineplugins = readLatestVersions(ctx, &o.repo)
		// writeback
		caches := []OfflinePlugin{}
		for _, v := range o.offlineplugins {
			log.Printf("found latest: %s-%s", v.Name, v.Version)
			caches = append(caches, OfflinePlugin{Name: v.Name, Version: v.Version})
		}
		if err := WritePluginFile(filename, caches); err != nil {
			return err
		}
	} else {
		o.offlineplugins = readCachedVersions(ctx, &o.repo, caches)
	}
	return nil
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

func readLatestVersions(ctx context.Context, repo *pluginmanager.Repository) map[string]*pluginmanager.PluginVersion {
	ret := map[string]*pluginmanager.PluginVersion{}
	for name, versions := range repo.Plugins {
		// do not download kubegems charts,it exists locally.
		if strings.HasPrefix(name, "kubegems") {
			continue
		}
		var cacheVersion *pluginmanager.PluginVersion
		// find latest version match kubegems
		for _, item := range versions {
			cacheVersion = &item
			break
		}
		if cacheVersion == nil {
			log.Printf("no matched version to cache on plugin %s", name)
			continue
		}
		ret[name] = cacheVersion
	}
	return ret
}

func (o *Offline) Download(ctx context.Context, basedir string) error {
	applier := bundle.NewDefaultApply(nil, nil, &bundle.Options{CacheDir: basedir})
	for _, pv := range o.offlineplugins {
		plugin := pv.ToPlugin()
		log.Printf("download %s-%s from %s", plugin.Name, plugin.Spec.Version, plugin.Spec.Kind)
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
	indexpath := bundle.PerRepoCacheDir(o.repo.Address, basedir)
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
	slices.SortFunc(list, func(a, b OfflinePlugin) bool {
		return strings.Compare(a.Name, b.Name) == -1
	})
	for _, val := range list {
		data.WriteString(val.Name + " " + val.Version + "\n")
	}
	return os.WriteFile(filename, data.Bytes(), helm.DefaultFileMode)
}

type OfflinePlugin struct {
	Name    string
	Version string
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
		case 0:
			continue
		case 1:
			ret = append(ret, OfflinePlugin{
				Name: string(fields[0]),
			})
		default:
			ret = append(ret, OfflinePlugin{
				Name:    string(fields[0]),
				Version: string(fields[1]),
			})
		}
	}
	return ret
}
