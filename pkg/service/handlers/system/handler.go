package systemhandler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/handlers/base"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/service/options"
)

type SystemHandler struct {
	base.BaseHandler
	*options.OnlineOptions
}

// GetConfig 列出所有系统配置
// @Tags System
// @Summary 列出所有系统配置
// @Description 列出所有系统配置
// @Accept json
// @Produce json
// @Success 200 {object} handlers.ResponseStruct{Data=[]models.OnlineConfig} "resp"
// @Router /v1/system/config [get]
// @Security JWT
func (h *SystemHandler) ListConfig(c *gin.Context) {
	cfgs := []models.OnlineConfig{}
	if err := h.GetDB().Find(&cfgs).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, cfgs)
}

// GetConfig 获取系统配置
// @Tags System
// @Summary 获取系统配置
// @Description 获取系统配置
// @Accept json
// @Produce json
// @Param name path string true "配置名"
// @Success 200 {object} handlers.ResponseStruct{Data=models.OnlineConfig} "resp"
// @Router /v1/system/config/{name} [get]
// @Security JWT
func (h *SystemHandler) GetConfig(c *gin.Context) {
	cfg := models.OnlineConfig{}
	if err := h.GetDB().First(&cfg, "name = ?", c.Param("name")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, cfg)
}

// SetConfig 修改系统配置
// @Tags System
// @Summary 修改系统配置
// @Description 修改系统配置
// @Accept json
// @Produce json
// @Param name path string true "配置名"
// @Param from body models.OnlineConfig true "配置内容"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "resp"
// @Router /v1/system/config/{name} [put]
// @Security JWT
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
	if err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&oldcfg).Error; err != nil {
			return err
		}
		return h.OnlineOptions.CheckAndUpdateSipecifiedField(oldcfg)
	}); err != nil {
		log.Error(err, "save config")
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

func (h *SystemHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/system/config", h.CheckIsSysADMIN, h.ListConfig)
	rg.GET("/system/config/:name", h.CheckIsSysADMIN, h.GetConfig)
	rg.PUT("/system/config/:name", h.CheckIsSysADMIN, h.SetConfig)
}
