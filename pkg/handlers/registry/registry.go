package registryhandler

import (
	"fmt"
	"strconv"

	"github.com/kubegems/gems/pkg/handlers"
	"github.com/kubegems/gems/pkg/models"

	"github.com/gin-gonic/gin"
)

var (
	SearchFields   = []string{"RegistryName"}
	FilterFields   = []string{"RegistryName"}
	PreloadFields  = []string{"Creator", "Project"}
	OrderFields    = []string{"RegistryName", "ID"}
	ModelName      = "Registry"
	PrimaryKeyName = "registry_id"
)

// ListRegistry 列表 Registry
// @Tags Registry
// @Summary Registry列表
// @Description Registry列表
// @Accept json
// @Produce json
// @Param RegistryName query string false "RegistryName"
// @Param preload query string false "choices Creator,Project"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (RegistryName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Registry}} "Registry"
// @Router /v1/registry [get]
// @Security JWT
func (h *RegistryHandler) ListRegistry(c *gin.Context) {
	var list []models.Registry
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         ModelName,
		SearchFields:  SearchFields,
		PreloadFields: PreloadFields,
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// RetrieveRegistry Registry详情
// @Tags Registry
// @Summary Registry详情
// @Description get Registry详情
// @Accept json
// @Produce json
// @Param registry_id path uint true "registry_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Registry} "Registry"
// @Router /v1/registry/{registry_id} [get]
// @Security JWT
func (h *RegistryHandler) RetrieveRegistry(c *gin.Context) {
	var obj models.Registry
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// PutRegistry 修改Registry
// @Tags Registry
// @Summary 修改Registry
// @Description 修改Registry
// @Accept json
// @Produce json
// @Param registry_id path uint true "registry_id"
// @Param param body models.Registry true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Registry} "Registry"
// @Router /v1/registry/{registry_id} [put]
// @Security JWT
func (h *RegistryHandler) PutRegistry(c *gin.Context) {
	var obj models.Registry
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "修改", "镜像仓库", obj.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, obj.ProjectID)
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if strconv.Itoa(int(obj.ID)) != c.Param(PrimaryKeyName) {
		handlers.NotOK(c, fmt.Errorf("请求体参数和URL参数ID不匹配"))
		return
	}

	// 检查其他默认仓库
	defaultRegistries := []models.Registry{}
	if err := h.GetDB().Where("project_id = ? and id != ? and is_default = ?", obj.ProjectID, obj.ID, true).
		Find(&defaultRegistries).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if len(defaultRegistries) > 0 && obj.IsDefault {
		handlers.NotOK(c, fmt.Errorf("默认仓库只能有一个"))
		return
	}
	if err := h.GetDB().Save(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, obj)
}

// DeleteRegistry 删除 Registry
// @Tags Registry
// @Summary 删除 Registry
// @Description 删除 Registry
// @Accept json
// @Produce json
// @Param registry_id path uint true "registry_id"
// @Success 204 {object} handlers.ResponseStruct "resp"
// @Router /v1/registry/{registry_id} [delete]
// @Security JWT
func (h *RegistryHandler) DeleteRegistry(c *gin.Context) {
	var obj models.Registry
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	h.SetAuditData(c, "删除", "镜像仓库", obj.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, obj.ProjectID)
	if err := h.GetDB().Delete(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.NoContent(c, nil)
}
