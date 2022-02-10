package systemhandler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/server/define"
	"kubegems.io/pkg/service/handlers"
)

type SystemHandler struct {
	define.ServerInterface
}

// GetConfig 获取系统配置
// @Tags System
// @Summary 获取系统配置
// @Description 获取系统配置
// @Accept json
// @Produce json
// @Param name path string true "配置名, metric"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Config} "Metrics配置"
// @Router /v1/system/config/{name} [get]
// @Security JWT
func (h *SystemHandler) GetConfig(c *gin.Context) {
	cfg := models.Config{}
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
// @Param name path string true "配置名, metric"
// @Param from body models.Config true "配置内容"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "Metrics配置"
// @Router /v1/system/config/{name} [put]
// @Security JWT
func (h *SystemHandler) SetConfig(c *gin.Context) {
	var oldcfg, newcfg models.Config
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

	oldcfg.Content = newcfg.Content
	if err := h.GetDB().Save(&oldcfg).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

func (h *SystemHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/system/config/:name", h.CheckIsSysADMIN, h.GetConfig)
	rg.PUT("/system/config/:name", h.CheckIsSysADMIN, h.SetConfig)
}
