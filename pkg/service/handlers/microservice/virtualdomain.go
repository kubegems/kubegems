package microservice

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/server/define"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
)

type VirtualDomainHandler struct {
	define.ServerInterface
}

// ListVirtualDomain 列表 VirtualDomain
// @Tags VirtualDomain
// @Summary VirtualDomain列表
// @Description VirtualDomain列表
// @Accept json
// @Produce json
// @Param VirtualDomainName query string false "VirtualDomainName"
// @Param VirtualDomainID query string false "VirtualDomainID"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (VirtualDomainName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.VirtualDomain}} "VirtualDomain"
// @Router /v1/virtualdomain [get]
// @Security JWT
func (h *VirtualDomainHandler) ListVirtualDomain(c *gin.Context) {
	var list []models.VirtualDomain
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:        "VirtualDomain",
		SearchFields: SearchFields,
		// Join:         handlers.Args("left join virtual_spaces on virtual_spaces.virtual_domain_id = virtual_domains.id"),
		// Select:       handlers.Args("virtual_domains.*, if(virtual_spaces.virtual_domain_id is null, false, true) as is_using"),
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// GetVirtualDomain VirtualDomain详情
// @Tags VirtualDomain
// @Summary VirtualDomain详情
// @Description get VirtualDomain详情
// @Accept json
// @Produce json
// @Param virtualdomain_id path uint true "virtualdomain_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.VirtualDomain} "VirtualDomain"
// @Router /v1/virtualdomain/{virtualdomain_id} [get]
// @Security JWT
func (h *VirtualDomainHandler) GetVirtualDomain(c *gin.Context) {
	// get vd
	vd := models.VirtualDomain{}
	if err := h.GetDB().First(&vd, c.Param("virtualdomain_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, vd)
}

// PostVirtualDomain 创建VirtualDomain
// @Tags VirtualDomain
// @Summary 创建VirtualDomain
// @Description 创建VirtualDomain
// @Accept json
// @Produce json
// @Param param body models.VirtualDomain true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.VirtualDomain} "VirtualDomain"
// @Router /v1/virtualdomain [post]
// @Security JWT
func (h *VirtualDomainHandler) PostVirtualDomain(c *gin.Context) {
	var vd models.VirtualDomain
	if err := c.BindJSON(&vd); err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "创建", "虚拟域名", vd.VirtualDomainName)
	h.SetExtraAuditData(c, models.ResVirtualDomain, vd.ID)

	u, _ := h.GetContextUser(c)
	vd.CreatedBy = u.Username
	vd.IsActive = true

	if err := h.GetDB().Save(&vd).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, vd)
}

// PutVirtualDomain 更新VirtualDomain
// @Tags VirtualDomain
// @Summary 更新VirtualDomain
// @Description 更新VirtualDomain
// @Accept json
// @Produce json
// @Param virtualdomain_id path uint true "virtualdomain_id"
// @Param param body models.VirtualDomain true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.VirtualDomain} "VirtualDomain"
// @Router /v1/virtualdomain/{virtualdomain_id} [put]
// @Security JWT
func (h *VirtualDomainHandler) PutVirtualDomain(c *gin.Context) {
	var obj models.VirtualDomain
	if err := h.GetDB().First(&obj, c.Param("virtualdomain_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "更新", "虚拟域名", obj.VirtualDomainName)
	h.SetExtraAuditData(c, models.ResTenant, obj.ID)

	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if strconv.Itoa(int(obj.ID)) != c.Param("virtualdomain_id") {
		handlers.NotOK(c, fmt.Errorf("数据ID错误"))
		return
	}
	if err := h.GetDB().Save(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// DeleteVirtualDomain 删除 VirtualDomain
// @Tags VirtualDomain
// @Summary 删除 VirtualDomain
// @Description 删除 VirtualDomain
// @Accept json
// @Produce json
// @Param virtualdomain_id path uint true "virtualdomain_id"
// @Success 200 {object} handlers.ResponseStruct "resp"
// @Router /v1/virtualdomain/{virtualdomain_id} [delete]
// @Security JWT
func (h *VirtualDomainHandler) DeleteVirtualDomain(c *gin.Context) {
	// get vd
	vd := models.VirtualDomain{}
	if err := h.GetDB().First(&vd, c.Param("virtualdomain_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "删除", "虚拟域名", vd.VirtualDomainName)
	h.SetExtraAuditData(c, models.ResVirtualDomain, vd.ID)

	if err := h.GetDB().Delete(&vd).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "")
}

// 为 service 设置 serviceentry
func (h *VirtualDomainHandler) InjectVirtualDomain(c *gin.Context) {}

// 为 service 取消设置 serviceentry
func (h *VirtualDomainHandler) UnInjectVirtualDomain(c *gin.Context) {}

func (h *VirtualDomainHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/virtualdomain", h.ListVirtualDomain)
	rg.POST("/virtualdomain", h.PostVirtualDomain)
	rg.GET("/virtualdomain/:virtualdomain_id", h.GetVirtualDomain)
	rg.PUT("/virtualdomain/:virtualdomain_id", h.PutVirtualDomain)
	rg.DELETE("/virtualdomain/:virtualdomain_id", h.DeleteVirtualDomain)
	rg.PUT("/virtualdomain/:virtualdomain_id/actions/inject", h.InjectVirtualDomain)
	rg.PUT("/virtualdomain/:virtualdomain_id/actions/uninject", h.InjectVirtualDomain)
}
