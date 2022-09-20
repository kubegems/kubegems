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

package registryhandler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
)

// PostProjectRegistry 创建一个属于 Project 的Registry
// @Tags        Project
// @Summary     创建一个属于 Project 的Registry
// @Description 创建一个属于 Project 的Registry
// @Accept      json
// @Produce     json
// @Param       project_id path     uint                                          true "project_id"
// @Param       param      body     models.Registry                               true "表单"
// @Success     200        {object} handlers.ResponseStruct{Data=models.Registry} "models.Registry"
// @Router      /v1/project/{project_id}/registry [post]
// @Security    JWT
func (h *RegistryHandler) PostProjectRegistry(c *gin.Context) {
	var registry models.Registry
	if err := c.BindJSON(&registry); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if strconv.Itoa(int(registry.ProjectID)) != c.Param("project_id") {
		handlers.NotOK(c, i18n.Errorf(c, "URL parameter mismatched with body"))
		return
	}

	// 检查默认仓库
	defaultRegistries := []models.Registry{}
	if err := h.GetDB().Where("project_id = ? and id != ? and is_default = ?", registry.ProjectID, registry.ID, true).
		Find(&defaultRegistries).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if len(defaultRegistries) > 0 && registry.IsDefault {
		handlers.NotOK(c, i18n.Errorf(c, "can't add image registry, there must be only one default image registry"))
		return
	}

	ctx := c.Request.Context()

	err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&registry).Error; err != nil {
			return err
		}
		return h.onChange(ctx, tx, &registry)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	action := i18n.Sprintf(context.TODO(), "add")
	module := i18n.Sprintf(context.TODO(), "image registry")
	h.SetAuditData(c, action, module, registry.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	handlers.Created(c, registry)
}

// ListProjectRegistry 获取属于Project的 Registry 列表
// @Tags        Project
// @Summary     获取属于 Project 的 Registry 列表
// @Description 获取属于 Project 的 Registry 列表
// @Accept      json
// @Produce     json
// @Param       project_id path     uint                                                                    true  "project_id"
// @Param       preload    query    string                                                                  false "choices Creator,Project"
// @Param       page       query    int                                                                     false "page"
// @Param       size       query    int                                                                     false "page"
// @Param       search     query    string                                                                  false "search in (RegistryName)"
// @Success     200        {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Registry}} "models.Registry"
// @Router      /v1/project/{project_id}/registry [get]
// @Security    JWT
func (h *RegistryHandler) ListProjectRegistry(c *gin.Context) {
	var list []models.Registry
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "Registry",
		SearchFields:  []string{"RegistryName"},
		PreloadFields: []string{"Creator", "Project"},
		Where:         []*handlers.QArgs{handlers.Args("project_id = ?", c.Param("project_id"))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// RetrieveProjectRegistry 获取Project 的一个 Registry详情
// @Tags        Project
// @Summary     获取Project 的一个 Registry详情
// @Description 获取Project 的一个 Registry详情
// @Accept      json
// @Produce     json
// @Param       project_id  path     uint                                          true "project_id"
// @Param       registry_id path     uint                                          true "registry_id"
// @Success     200         {object} handlers.ResponseStruct{Data=models.Registry} "models.Registry"
// @Router      /v1/project/{project_id}/registry/{registry_id} [get]
// @Security    JWT
func (h *RegistryHandler) RetrieveProjectRegistry(c *gin.Context) {
	var registry models.Registry
	if err := h.GetDB().First(&registry, "project_id = ? and id = ?", c.Param(ProjectKeyName), c.Param("registry_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, registry)
}

// PutProjectRegistry 修改 Project 的 Registry
// @Tags        Project
// @Summary     修改Project 的 Registry
// @Description 修改 Project 的 Registry
// @Accept      json
// @Produce     json
// @Param       project_id  path     uint                                          true "project_id"
// @Param       registry_id path     uint                                          true "registry_id"
// @Success     200         {object} handlers.ResponseStruct{Data=models.Registry} "models.Registry"
// @Router      /v1/project/{project_id}/registry/{registry_id} [put]
// @Security    JWT
func (h *RegistryHandler) PutProjectRegistry(c *gin.Context) {
	var registry models.Registry
	if err := h.GetDB().First(&registry, "project_id = ? and id = ?", c.Param(ProjectKeyName), c.Param("registry_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := c.Bind(&registry); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if strconv.Itoa(int(registry.ProjectID)) != c.Param(ProjectKeyName) || strconv.Itoa(int(registry.ID)) != c.Param("registry_id") {
		handlers.NotOK(c, i18n.Errorf(c, "URL parameter mismatched with body"))
		return
	}

	// 检查默认仓库
	defaultRegistries := []models.Registry{}
	if err := h.GetDB().Where("project_id = ? and id != ? and is_default = ?", registry.ProjectID, registry.ID, true).
		Find(&defaultRegistries).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if len(defaultRegistries) > 0 && registry.IsDefault {
		handlers.NotOK(c, i18n.Errorf(c, "can't update image registry, the default image registry can only exist one"))
		return
	}

	ctx := c.Request.Context()
	err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&registry).Error; err != nil {
			return err
		}
		return h.onChange(ctx, tx, &registry)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	action := i18n.Sprintf(context.TODO(), "update")
	module := i18n.Sprintf(context.TODO(), "image registry")
	h.SetAuditData(c, action, module, registry.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	handlers.OK(c, registry)
}

// SetDefaultProjectRegistry 设置 Project 的 默认 Registry
// @Tags        Project
// @Summary     设置 Project 的 默认 Registry
// @Description 设置 Project 的 默认 Registry
// @Accept      json
// @Produce     json
// @Param       project_id  path     uint                                          true "project_id"
// @Param       registry_id path     uint                                          true "registry_id"
// @Param       is_default  query    bool                                          true "是否默认镜像仓库"
// @Success     200         {object} handlers.ResponseStruct{Data=models.Registry} "models.Registry"
// @Router      /v1/project/{project_id}/registry/{registry_id} [patch]
// @Security    JWT
func (h *RegistryHandler) SetDefaultProjectRegistry(c *gin.Context) {
	var registry models.Registry
	if err := h.GetDB().First(&registry, "project_id = ? and id = ?", c.Param(ProjectKeyName), c.Param("registry_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	isDefault, _ := strconv.ParseBool(c.Query("isDefault"))

	registry.IsDefault = isDefault

	action := ""
	module := i18n.Sprintf(context.TODO(), "the project's default image registry ")
	if isDefault {
		action = i18n.Sprintf(context.TODO(), "set")
	} else {
		action = i18n.Sprintf(context.TODO(), "unset")
	}
	h.SetAuditData(c, action, module, registry.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	// 检查默认仓库
	defaultRegistries := []models.Registry{}
	if err := h.GetDB().Where("project_id = ? and id != ? and is_default = ?", registry.ProjectID, registry.ID, true).
		Find(&defaultRegistries).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if len(defaultRegistries) > 0 && registry.IsDefault {
		handlers.NotOK(c, i18n.Errorf(c, "can't set image registry, the project's default image registry can only exist one"))
		return
	}

	ctx := c.Request.Context()
	err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&registry).Error; err != nil {
			return err
		}
		return h.onChange(ctx, tx, &registry)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	handlers.OK(c, registry)
}

// DeleteProjectRegistry Project 的 Registry
// @Tags        Project
// @Summary     删除 Project 的 Registry
// @Description 删除 Project 的 Registry
// @Accept      json
// @Produce     json
// @Param       project_id  path     uint                                          true "project_id"
// @Param       registry_id path     uint                                          true "registry_id"
// @Success     200         {object} handlers.ResponseStruct{Data=models.Registry} "models.Registry"
// @Router      /v1/project/{project_id}/registry/{registry_id} [delete]
// @Security    JWT
func (h *RegistryHandler) DeleteProjectRegistry(c *gin.Context) {
	var registry models.Registry
	h.GetDB().First(&registry, c.Param("registry_id"))

	ctx := c.Request.Context()
	err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&registry, "project_id = ? and id = ?", c.Param(ProjectKeyName), c.Param("registry_id")).Error; err != nil {
			return err
		}
		return h.onDelete(ctx, tx, &registry)
	})
	if err != nil {
		handlers.NotOK(c, i18n.Errorf(c, "failed to delete the image registry: %v", err))
	}

	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "image registry")
	h.SetAuditData(c, action, module, registry.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	handlers.NoContent(c, nil)
}
