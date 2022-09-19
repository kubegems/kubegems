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
	"strconv"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/installer/api"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/gemsplugin"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
)

type PluginHandler struct {
	PM *gemsplugin.PluginManager
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

type MainCategory map[string]map[string][]api.PluginStatus

// @Tags        Agent.Plugin
// @Summary     获取Plugin列表数据
// @Description 获取Plugin列表数据
// @Accept      json
// @Produce     json
// @Param       cluster path     string                                                                 true "cluster"
// @Param       simple  query    bool                                                                   true "simple"
// @Success     200     {object} handlers.ResponseStruct{Data=map[string]map[string][]api.PluginStatus} "Plugins"
// @Router      /v1/proxy/cluster/{cluster}/plugins [get]
// @Security    JWT
func (h *PluginHandler) List(c *gin.Context) {
	plugins, err := h.PM.ListPlugins(c.Request.Context())
	if err != nil {
		NotOK(c, err)
		return
	}
	if simple, _ := strconv.ParseBool(c.Query("simple")); simple {
		ret := map[string]bool{}
		for name, v := range plugins {
			ret[name] = (v.Installed != nil)
		}
		OK(c, ret)
		return
	} else {
		categoriedPlugins := api.CategoriedPlugins(plugins)
		OK(c, categoriedPlugins)
	}
}

// @Tags        Agent.Plugin
// @Summary     插件详情
// @Description 插件详情
// @Accept      json
// @Produce     json
// @Param       cluster path     string                                                 true "cluster"
// @Param       name    path     string                                                 true "name"
// @Param       version query    string                                                 true "version"
// @Success     200     {object} handlers.ResponseStruct{Data=gemsplugin.PluginVersion} "Plugins"
// @Router      /v1/proxy/cluster/{cluster}/plugins/{name} [get]
// @Security    JWT
func (h *PluginHandler) Get(c *gin.Context) {
	name, version := c.Param("name"), c.Query("version")
	plugin, err := h.PM.GetPluginVersion(c.Request.Context(), name, version, true)
	if err != nil {
		NotOK(c, err)
		return
	}
	OK(c, plugin)
}

// @Tags        Agent.Plugin
// @Summary     启用插件
// @Description 启用插件
// @Accept      json
// @Produce     json
// @Param       cluster path     string                               true "cluster"
// @Param       name    path     string                               true "name"
// @Param       body    body     gemsplugin.PluginVersion             true "pluginVersion"
// @Success     200     {object} handlers.ResponseStruct{Data=string} "ok"
// @Router      /v1/proxy/cluster/{cluster}/plugins/{name} [post]
// @Security    JWT
func (h *PluginHandler) Enable(c *gin.Context) {
	name := c.Param("name")

	pv := gemsplugin.PluginVersion{}
	if err := request.Body(c.Request, &pv); err != nil {
		NotOK(c, err)
		return
	}

	if err := h.PM.Install(c.Request.Context(), name, pv.Version, pv.Values.Object); err != nil {
		log.Error(err, "update plugin", "plugin", c.Param("name"))
		NotOK(c, err)
		return
	}
	OK(c, pv)
}

// @Tags        Agent.Plugin
// @Summary     禁用插件
// @Description 禁用插件
// @Accept      json
// @Produce     json
// @Param       cluster path     string                               true "cluster"
// @Param       name    path     string                               true "name"
// @Success     200     {object} handlers.ResponseStruct{Data=string} "Plugins"
// @Router      /v1/proxy/cluster/{cluster}/plugins [delete]
// @Security    JWT
func (h *PluginHandler) Disable(c *gin.Context) {
	name := c.Param("name")

	if err := h.PM.UnInstall(c.Request.Context(), name); err != nil {
		log.Error(err, "update plugin", "plugin", c.Param("name"))
		NotOK(c, err)
		return
	}
	OK(c, "ok")
}

// @Tags        Agent.Plugin
// @Summary     检查更新
// @Description 检查更新
// @Accept      json
// @Produce     json
// @Param       cluster path     string                                           true "cluster"
// @Success     200     {object} handlers.ResponseStruct{Data=[]api.PluginStatus} "ok"
// @Router      /v1/proxy/cluster/{cluster}/plugins:check-update [post]
func (h *PluginHandler) CheckUpdate(c *gin.Context) {
	upgradeable, err := h.PM.CheckUpdate(c.Request.Context())
	if err != nil {
		NotOK(c, err)
		return
	}

	upgradeableStatus := []api.PluginStatus{}
	for _, item := range upgradeable {
		upgradeableStatus = append(upgradeableStatus, api.ToViewPlugin(item))
	}
	api.SortPluginStatusByName(upgradeableStatus)
	OK(c, upgradeableStatus)
}
