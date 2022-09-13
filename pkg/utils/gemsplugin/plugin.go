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

package gemsplugin

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"golang.org/x/exp/slices"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pluginscommon "kubegems.io/kubegems/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/bundle"
	"kubegems.io/kubegems/pkg/installer/bundle/helm"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const DefaultPluginsDir = "plugins"

type Plugin struct {
	Name         string          `json:"name"`
	Namespace    string          `json:"namespace"`
	MainCategory string          `json:"mainCategory"`
	Category     string          `json:"category"`
	Upgradeable  *PluginVersion  `json:"upgradeable"`
	Required     bool            `json:"required"`
	Installed    *PluginVersion  `json:"installed"`
	Available    []PluginVersion `json:"available"`
	Description  string          `json:"description"`
}

type PluginManager struct {
	CacheDir string
	Client   client.Client
}

func DefaultPluginManager(cachedir string) (*PluginManager, error) {
	cfg, err := kube.AutoClientConfig()
	if err != nil {
		return nil, err
	}
	cli, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}
	return &PluginManager{CacheDir: cachedir, Client: cli}, nil
}

func (m *PluginManager) Install(ctx context.Context, name string, version string, values map[string]any) error {
	pv, err := m.GetPluginVersion(ctx, name, version, false)
	if err != nil {
		return err
	}
	pv.Values = pluginsv1beta1.Values{Object: values}.FullFill()
	apiplugin := pv.ToPlugin()
	// all of plugins must install in installer namespace
	apiplugin.Namespace = pluginscommon.KubeGemsNamespaceInstaller

	exists := apiplugin.DeepCopy()
	_, err = controllerutil.CreateOrUpdate(ctx, m.Client, apiplugin, func() error {
		exists.Annotations = apiplugin.Annotations
		exists.Spec = apiplugin.Spec
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *PluginManager) UnInstall(ctx context.Context, name string) error {
	return m.Client.Delete(ctx, &pluginsv1beta1.Plugin{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: pluginscommon.KubeGemsNamespaceInstaller,
		},
	})
}

func (m *PluginManager) Get(ctx context.Context, name string) (*Plugin, error) {
	installed, _ := m.GetInstalled(ctx, name)
	remotes, _ := m.GetRemote(ctx, name)

	if installed == nil && len(remotes) == 0 {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	showVersion := PluginVersion{Name: name}
	if installed != nil {
		showVersion = *installed
	} else if len(remotes) != 0 {
		showVersion = remotes[0]
	}
	var upgradeVersion *PluginVersion
	if installed != nil {
		for _, remote := range remotes {
			if semver.Compare(remote.Version, installed.Version) > -1 {
				upgradeVersion = &remote
			}
		}
	}
	plugin := Plugin{
		Name:         name,
		Installed:    installed,
		Available:    remotes,
		MainCategory: showVersion.MainCategory,
		Category:     showVersion.Category,
		Upgradeable:  upgradeVersion,
		Description:  showVersion.Description,
	}
	return &plugin, nil
}

func (m *PluginManager) GetPluginVersion(ctx context.Context, name, version string, withSchema bool) (*PluginVersion, error) {
	plugin, err := m.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	allversions := plugin.Available
	if plugin.Installed != nil {
		// if no version speci always return installed version
		allversions = append([]PluginVersion{*plugin.Installed}, allversions...)
	}
	for _, pv := range allversions {
		// if  no version we use the first version
		// nolint: nestif
		if version == "" || pv.Version == version {
			// find schema
			if withSchema {
				if err := m.fillSchema(ctx, &pv); err != nil {
					logr.FromContextOrDiscard(ctx).Error(err, "get schema", "plugin", pv.Name, "version", pv.Version)
				}
			}
			return &pv, nil
		}
	}
	return nil, fmt.Errorf("plugin %s version %s not found", name, version)
}

func (m *PluginManager) fillSchema(ctx context.Context, pv *PluginVersion) error {
	if m.CacheDir == "" {
		m.CacheDir = DefaultPluginsDir
	}
	// we cache in a dir same with plugins use.
	cachedir := bundle.PerRepoCacheDir(pv.Repository, m.CacheDir)
	_, chart, err := helm.Download(ctx, pv.Repository, pv.Name, pv.Version, cachedir)
	if err != nil {
		return err
	} else {
		pv.Schema = string(chart.Schema)
		return nil
	}
}

func (m *PluginManager) ListPlugins(ctx context.Context) (map[string]Plugin, error) {
	// list local
	installversions, err := m.ListInstalled(ctx, false)
	if err != nil {
		return nil, err
	}
	// list remotes
	avaliableversions, err := m.ListRemote(ctx)
	if err != nil {
		return nil, err
	}

	fillmaindesc := func(p *Plugin, pv PluginVersion) {
		p.Description = pv.Description
		p.MainCategory = pv.MainCategory
		p.Category = pv.Category
		p.Required = pv.Required
		p.Namespace = pv.Namespace
	}

	plugins := map[string]Plugin{}
	for name, available := range avaliableversions {
		if len(available) == 0 {
			continue
		}
		p := Plugin{
			Name:      name,
			Available: available,
		}
		if installed, ok := installversions[name]; ok {
			p.Installed = &installed
			// remove from map we added it to plugin.
			delete(installversions, name)
			p.Upgradeable = FindUpgradeable(available, installed, installversions) // check upgrade
			fillmaindesc(&p, installed)
		} else {
			fillmaindesc(&p, available[0])
		}
		plugins[name] = p
	}
	// installed but not in remotes
	for name, val := range installversions {
		installed := val
		p := Plugin{
			Name:      name,
			Installed: &installed,
		}
		fillmaindesc(&p, installed)
		plugins[name] = p
	}
	return plugins, nil
}

func FindUpgradeable(availables []PluginVersion, installed PluginVersion, allinstall map[string]PluginVersion) *PluginVersion {
	for _, available := range availables {
		if !SemVersionBiggerThan(available.Version, installed.Version) {
			continue
		}
		// meet all requirements
		if CheckDependecies(available.Requirements, allinstall) == nil {
			return &available
		}
	}
	return nil
}

func (m *PluginManager) GetInstalled(ctx context.Context, name string) (*PluginVersion, error) {
	plugin := &pluginsv1beta1.Plugin{}
	if err := m.Client.Get(ctx,
		client.ObjectKey{Namespace: pluginscommon.KubeGemsNamespaceInstaller, Name: name},
		plugin,
	); err != nil {
		return nil, err
	}
	pv := PluginVersionFrom(plugin)
	return &pv, nil
}

func (m *PluginManager) ListInstalled(ctx context.Context, checkHealthy bool) (map[string]PluginVersion, error) {
	pluginList := &pluginsv1beta1.PluginList{}
	if err := m.Client.List(ctx, pluginList, client.InNamespace(pluginscommon.KubeGemsNamespaceInstaller)); err != nil {
		return nil, err
	}
	ret := map[string]PluginVersion{}
	for _, plugin := range pluginList.Items {
		ret[plugin.Name] = PluginVersionFrom(&plugin)
	}
	if checkHealthy {
		eg := errgroup.Group{}
		for name := range ret {
			name := name
			eg.Go(func() error {
				pv := ret[name]
				CheckHealthy(ctx, m.Client, &pv)
				ret[name] = pv
				return nil
			})
		}
		eg.Wait()
	}
	return ret, nil
}

func (m *PluginManager) GetRemote(ctx context.Context, name string) ([]PluginVersion, error) {
	repos, err := m.ListRepos(ctx)
	if err != nil {
		return nil, err
	}
	ret := []PluginVersion{}
	for _, repo := range repos {
		ret = append(ret, repo.Plugins[name]...)
	}
	slices.SortFunc(ret, func(a, b PluginVersion) bool {
		return semver.Compare(a.Version, b.Version) > -1
	})
	return ret, nil
}

func (m *PluginManager) ListRemote(ctx context.Context) (map[string][]PluginVersion, error) {
	repos, err := m.ListRepos(ctx)
	if err != nil {
		return nil, err
	}
	ret := map[string][]PluginVersion{}
	for _, repo := range repos {
		for name, pvs := range repo.Plugins {
			if pluginversions, ok := ret[name]; ok {
				ret[name] = append(pluginversions, pvs...)
			} else {
				ret[name] = pvs
			}
		}
	}
	for name := range ret {
		slices.SortFunc(ret[name], func(a, b PluginVersion) bool {
			return semver.Compare(a.Version, b.Version) > -1
		})
	}
	return ret, nil
}

func (m *PluginManager) CheckUpdate(ctx context.Context) (map[string]Plugin, error) {
	// update repo index
	if err := m.UpdateReposCache(ctx); err != nil {
		return nil, err
	}

	// list plugins
	plugins, err := m.ListPlugins(ctx)
	if err != nil {
		return nil, err
	}

	// filter upgradable
	upgradable := map[string]Plugin{}
	for name, plugin := range plugins {
		if plugin.Upgradeable != nil {
			upgradable[name] = plugin
		}
	}
	return upgradable, nil
}
