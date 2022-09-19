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

package api

import (
	"sort"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/utils/gemsplugin"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

type PluginStatus struct {
	Name               string         `json:"name"`
	Namespace          string         `json:"namespace"`
	Description        string         `json:"description"`
	InstalledVersion   string         `json:"installedVersion"`
	UpgradeableVersion string         `json:"upgradeableVersion"`
	AvailableVersions  []string       `json:"availableVersions"`
	Required           bool           `json:"required"`
	Enabled            bool           `json:"enabled"`
	Healthy            bool           `json:"healthy"`
	Message            string         `json:"message"`
	Values             map[string]any `json:"values"` // current installed version values
	maincate           string
	cate               string
}

func (o *PluginsAPI) ListPlugins(req *restful.Request, resp *restful.Response) {
	plugins, err := o.PM.ListPlugins(req.Request.Context())
	if err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, CategoriedPlugins(plugins))
}

func SortPluginStatusByName(list []PluginStatus) {
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
}

func ToViewPlugin(plugin gemsplugin.Plugin) PluginStatus {
	ps := PluginStatus{
		Name:        plugin.Name,
		Namespace:   plugin.Namespace,
		Description: plugin.Description,
		Required:    plugin.Required,
		maincate:    plugin.MainCategory,
		cate:        plugin.Category,
	}
	if installed := plugin.Installed; installed != nil {
		ps.InstalledVersion = installed.Version
		ps.Healthy = installed.Healthy
		ps.Namespace = installed.InstallNamespace
		ps.Enabled = true
		ps.Values = installed.Values.Object
	}
	if upgradble := plugin.Upgradeable; upgradble != nil {
		ps.UpgradeableVersion = upgradble.Version
	}

	availableVersion := []string{}
	for _, item := range plugin.Available {
		availableVersion = append(availableVersion, item.Version)
	}
	ps.AvailableVersions = availableVersion
	return ps
}

func CategoriedPlugins(plugins map[string]gemsplugin.Plugin) map[string]map[string][]PluginStatus {
	pluginstatus := []PluginStatus{}

	for _, plugin := range plugins {
		pluginstatus = append(pluginstatus, ToViewPlugin(plugin))
	}

	mainCategoryFunc := func(t PluginStatus) string {
		return t.maincate
	}
	categoryfunc := func(t PluginStatus) string {
		return t.cate
	}

	categoryPlugins := map[string]map[string][]PluginStatus{}
	for maincategory, list := range withCategory(pluginstatus, mainCategoryFunc) {
		categorized := withCategory(list, categoryfunc)
		// sort
		for _, list := range categorized {
			SortPluginStatusByName(list)
		}
		categoryPlugins[maincategory] = categorized
	}
	return categoryPlugins
}

func withCategory[T any](list []T, getCate func(T) string) map[string][]T {
	ret := map[string][]T{}
	for _, v := range list {
		cate := getCate(v)
		if cate == "" {
			cate = "others"
		}
		if _, ok := ret[cate]; !ok {
			ret[cate] = []T{}
		}
		ret[cate] = append(ret[cate], v)
	}
	return ret
}

func (o *PluginsAPI) GetPlugin(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	version := req.QueryParameter("version")

	pv, err := o.PM.GetPluginVersion(req.Request.Context(), name, version, true)
	if err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, pv)
}

func (o *PluginsAPI) EnablePlugin(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	version := req.QueryParameter("version")

	pv := &gemsplugin.PluginVersion{}
	if err := request.Body(req.Request, pv); err != nil {
		response.Error(resp, err)
		return
	}
	if err := o.PM.Install(req.Request.Context(), name, version, pv.Values.Object); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, pv)
}

func (o *PluginsAPI) RemovePlugin(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	if err := o.PM.UnInstall(req.Request.Context(), name); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, "ok")
}
