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

package apis

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/agent/cluster"
	pluginscommon "kubegems.io/kubegems/pkg/apis/plugins"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils/gemsplugin"
)

type PluginHandler struct {
	cluster cluster.Interface
}

type PluginStatus struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Required     bool   `json:"required"`
	Icon         string `json:"icon"`
	Description  string `json:"description"`
	Version      string `json:"version"`
	Enabled      bool   `json:"enabled"`
	Healthy      bool   `json:"healthy"`
	Message      string `json:"message"`
	mainCategory string `json:"-"`
	category     string `json:"-"`
}

// core -> category
type MainCategory map[string]CategoriedPlugins

type SimplePlugin map[string]bool

// category -> plugin
type CategoriedPlugins map[string][]PluginStatus

// @Tags        Agent.Plugin
// @Summary     获取Plugin列表数据
// @Description 获取Plugin列表数据
// @Accept      json
// @Produce     json
// @Param       cluster path     string                                     true "cluster"
// @Param       simple  query    bool                                       true "simple"
// @Success     200     {object} handlers.ResponseStruct{Data=MainCategory} "Plugins"
// @Router      /v1/proxy/cluster/{cluster}/custom/plugins.kubegems.io/v1beta1/installers [get]
// @Security    JWT
func (h *PluginHandler) List(c *gin.Context) {
	_, plugins, err := gemsplugin.ListPlugins(c.Request.Context(), h.cluster.GetClient())
	if err != nil {
		NotOK(c, err)
		return
	}
	// convert to view model
	viewplugins := make([]PluginStatus, 0, len(plugins))
	for _, plugin := range plugins {
		viewplugin := PluginStatus{
			Name:      plugin.Name,
			Namespace: plugin.Namespace,
			Version:   plugin.Version,
			Enabled:   plugin.Enabled,
			Healthy:   plugin.Healthy,
			Message:   plugin.Message,
		}
		if annotaions := plugin.Annotations; annotaions != nil {
			viewplugin.mainCategory = annotaions[pluginscommon.AnnotationMainCategory]
			viewplugin.category = annotaions[pluginscommon.AnnotationCategory]
			viewplugin.Description = annotaions[pluginscommon.AnnotationDescription]
			viewplugin.Icon = annotaions[pluginscommon.AnnotationIcon]
			viewplugin.Required, _ = strconv.ParseBool(annotaions[pluginscommon.AnnotationRequired])
		}
		viewplugins = append(viewplugins, viewplugin)
	}
	if simple, _ := strconv.ParseBool(c.Query("simple")); simple {
		ret := SimplePlugin{}
		for _, v := range viewplugins {
			ret[v.Name] = v.Healthy
		}
		OK(c, ret)
		return
	}
	mainCategoryFunc := func(t PluginStatus) string {
		return t.mainCategory
	}
	categoryfunc := func(t PluginStatus) string {
		return t.category
	}
	categoryPlugins := MainCategory{}
	for maincategory, list := range withCategory(viewplugins, mainCategoryFunc) {
		categorized := withCategory(list, categoryfunc)
		// sort
		for _, list := range categorized {
			sort.Slice(list, func(i, j int) bool {
				return list[i].Name < list[j].Name
			})
		}
		categoryPlugins[maincategory] = CategoriedPlugins(categorized)
	}
	OK(c, categoryPlugins)
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

// @Tags        Agent.Plugin
// @Summary     启用插件
// @Description 启用插件
// @Accept      json
// @Produce     json
// @Param       cluster path     string                               true "cluster"
// @Param       name    path     string                               true "name"
// @Param       type    query    string                               true "type"
// @Success     200     {object} handlers.ResponseStruct{Data=string} "Plugins"
// @Router      /v1/proxy/cluster/{cluster}/custom/plugins.kubegems.io/v1beta1/installers/{name}/actions/enable [put]
// @Security    JWT
func (h *PluginHandler) Enable(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		handlers.NotOK(c, fmt.Errorf("empty plugin name"))
		return
	}
	if err := gemsplugin.EnablePlugin(c.Request.Context(), h.cluster.GetClient(), name, true); err != nil {
		log.Error(err, "update plugin", "plugin", c.Param("name"))
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// @Tags        Agent.Plugin
// @Summary     禁用插件
// @Description 禁用插件
// @Accept      json
// @Produce     json
// @Param       cluster path     string                               true "cluster"
// @Param       name    path     string                               true "name"
// @Param       type    query    string                               true "type"
// @Success     200     {object} handlers.ResponseStruct{Data=string} "Plugins"
// @Router      /v1/proxy/cluster/{cluster}/custom/plugins.kubegems.io/v1beta1/installers/{name}/actions/disable [put]
// @Security    JWT
func (h *PluginHandler) Disable(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		handlers.NotOK(c, fmt.Errorf("empty plugin name"))
		return
	}
	if err := gemsplugin.EnablePlugin(c.Request.Context(), h.cluster.GetClient(), name, false); err != nil {
		log.Error(err, "update plugin", "plugin", c.Param("name"))
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}
