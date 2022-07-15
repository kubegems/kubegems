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

package systemhandler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	"kubegems.io/kubegems/pkg/service/models"
)

type SystemHandler struct {
	base.BaseHandler
}

// GetConfig 列出所有系统配置
// @Tags         System
// @Summary      列出所有系统配置
// @Description  列出所有系统配置
// @Accept       json
// @Produce      json
// @Success      200  {object}  handlers.ResponseStruct{Data=[]models.OnlineConfig}  "resp"
// @Router       /v1/system/config [get]
// @Security     JWT
func (h *SystemHandler) ListConfig(c *gin.Context) {
	cfgs := []models.OnlineConfig{}
	if err := h.GetDB().Find(&cfgs).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, cfgs)
}

// GetConfig 获取系统配置
// @Tags         System
// @Summary      获取系统配置
// @Description  获取系统配置
// @Accept       json
// @Produce      json
// @Param        name  path      string                                             true  "配置名"
// @Success      200   {object}  handlers.ResponseStruct{Data=models.OnlineConfig}  "resp"
// @Router       /v1/system/config/{name} [get]
// @Security     JWT
func (h *SystemHandler) GetConfig(c *gin.Context) {
	cfg := models.OnlineConfig{}
	if err := h.GetDB().First(&cfg, "name = ?", c.Param("name")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, cfg)
}

// SetConfig 修改系统配置
// @Tags         System
// @Summary      修改系统配置
// @Description  修改系统配置
// @Accept       json
// @Produce      json
// @Param        name  path      string                                true  "配置名"
// @Param        from  body      models.OnlineConfig                   true  "配置内容"
// @Success      200   {object}  handlers.ResponseStruct{Data=string}  "resp"
// @Router       /v1/system/config/{name} [put]
// @Security     JWT
func (h *SystemHandler) SetConfig(c *gin.Context) {
	var oldcfg, newcfg models.OnlineConfig
	if err := h.GetDB().First(&oldcfg, "name = ?", c.Param("name")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := c.BindJSON(&newcfg); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if oldcfg.Name != newcfg.Name {
		handlers.NotOK(c, fmt.Errorf("配置名不一致"))
		return
	}

	h.SetAuditData(c, "更新", "系统配置", newcfg.Name)

	oldcfg.Content = newcfg.Content
	if err := h.GetDB().Save(&oldcfg).Error; err != nil {
		log.Error(err, "save config")
		handlers.NotOK(c, err)
	}
	handlers.OK(c, "ok")
}

func (h *SystemHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/system/config", h.CheckIsSysADMIN, h.ListConfig)
	rg.GET("/system/config/:name", h.CheckIsSysADMIN, h.GetConfig)
	rg.PUT("/system/config/:name", h.CheckIsSysADMIN, h.SetConfig)
}
