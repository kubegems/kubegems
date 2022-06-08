package registryhandler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
)

// PostProjectRegistry 创建一个属于 Project 的Registry
// @Tags         Project
// @Summary      创建一个属于 Project 的Registry
// @Description  创建一个属于 Project 的Registry
// @Accept       json
// @Produce      json
// @Param        project_id  path      uint                                           true  "project_id"
// @Param        param       body      models.Registry                                true  "表单"
// @Success      200         {object}  handlers.ResponseStruct{Data=models.Registry}  "models.Registry"
// @Router       /v1/project/{project_id}/registry [post]
// @Security     JWT
func (h *RegistryHandler) PostProjectRegistry(c *gin.Context) {
	var registry models.Registry
	if err := c.BindJSON(&registry); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if strconv.Itoa(int(registry.ProjectID)) != c.Param("project_id") {
		handlers.NotOK(c, fmt.Errorf("项目id不一致"))
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
		handlers.NotOK(c, fmt.Errorf("默认仓库只能有一个"))
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

	h.SetAuditData(c, "创建", "镜像仓库", registry.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	handlers.Created(c, registry)
}

// ListProjectRegistry 获取属于Project的 Registry 列表
// @Tags         Project
// @Summary      获取属于 Project 的 Registry 列表
// @Description  获取属于 Project 的 Registry 列表
// @Accept       json
// @Produce      json
// @Param        project_id  path      uint                                                                     true   "project_id"
// @Param        preload     query     string                                                                   false  "choices Creator,Project"
// @Param        page        query     int                                                                      false  "page"
// @Param        size        query     int                                                                      false  "page"
// @Param        search      query     string                                                                   false  "search in (RegistryName)"
// @Success      200         {object}  handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Registry}}  "models.Registry"
// @Router       /v1/project/{project_id}/registry [get]
// @Security     JWT
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
// @Tags         Project
// @Summary      获取Project 的一个 Registry详情
// @Description  获取Project 的一个 Registry详情
// @Accept       json
// @Produce      json
// @Param        project_id   path      uint                                           true  "project_id"
// @Param        registry_id  path      uint                                           true  "registry_id"
// @Success      200          {object}  handlers.ResponseStruct{Data=models.Registry}  "models.Registry"
// @Router       /v1/project/{project_id}/registry/{registry_id} [get]
// @Security     JWT
func (h *RegistryHandler) RetrieveProjectRegistry(c *gin.Context) {
	var registry models.Registry
	if err := h.GetDB().First(&registry, "project_id = ? and id = ?", c.Param(ProjectKeyName), c.Param("registry_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, registry)
}

// PutProjectRegistry 修改 Project 的 Registry
// @Tags         Project
// @Summary      修改Project 的 Registry
// @Description  修改 Project 的 Registry
// @Accept       json
// @Produce      json
// @Param        project_id   path      uint                                           true  "project_id"
// @Param        registry_id  path      uint                                           true  "registry_id"
// @Success      200          {object}  handlers.ResponseStruct{Data=models.Registry}  "models.Registry"
// @Router       /v1/project/{project_id}/registry/{registry_id} [put]
// @Security     JWT
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
		handlers.NotOK(c, fmt.Errorf("请求体参数和URL参数不匹配"))
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
		handlers.NotOK(c, fmt.Errorf("默认仓库只能有一个"))
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

	h.SetAuditData(c, "更新", "镜像仓库", registry.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	handlers.OK(c, registry)
}

// SetDefaultProjectRegistry 设置 Project 的 默认 Registry
// @Tags         Project
// @Summary      设置 Project 的 默认 Registry
// @Description  设置 Project 的 默认 Registry
// @Accept       json
// @Produce      json
// @Param        project_id   path      uint                                           true  "project_id"
// @Param        registry_id  path      uint                                           true  "registry_id"
// @Param        is_default   query     bool                                           true  "是否默认镜像仓库"
// @Success      200          {object}  handlers.ResponseStruct{Data=models.Registry}  "models.Registry"
// @Router       /v1/project/{project_id}/registry/{registry_id} [patch]
// @Security     JWT
func (h *RegistryHandler) SetDefaultProjectRegistry(c *gin.Context) {
	var registry models.Registry
	if err := h.GetDB().First(&registry, "project_id = ? and id = ?", c.Param(ProjectKeyName), c.Param("registry_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	isDefault, _ := strconv.ParseBool(c.Query("isDefault"))

	registry.IsDefault = isDefault

	if isDefault {
		h.SetAuditData(c, "设置默认", "镜像仓库", registry.RegistryName)
	} else {
		h.SetAuditData(c, "取消默认", "镜像仓库", registry.RegistryName)
	}
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	// 检查默认仓库
	defaultRegistries := []models.Registry{}
	if err := h.GetDB().Where("project_id = ? and id != ? and is_default = ?", registry.ProjectID, registry.ID, true).
		Find(&defaultRegistries).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if len(defaultRegistries) > 0 && registry.IsDefault {
		handlers.NotOK(c, fmt.Errorf("默认仓库只能有一个"))
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
// @Tags         Project
// @Summary      删除 Project 的 Registry
// @Description  删除 Project 的 Registry
// @Accept       json
// @Produce      json
// @Param        project_id   path      uint                                           true  "project_id"
// @Param        registry_id  path      uint                                           true  "registry_id"
// @Success      200          {object}  handlers.ResponseStruct{Data=models.Registry}  "models.Registry"
// @Router       /v1/project/{project_id}/registry/{registry_id} [delete]
// @Security     JWT
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
		handlers.NotOK(c, fmt.Errorf("删除仓库错误 %v", err))
	}

	h.SetAuditData(c, "删除", "镜像仓库", registry.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, registry.ProjectID)

	handlers.NoContent(c, nil)
}
