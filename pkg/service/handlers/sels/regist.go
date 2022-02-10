package sels

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/server/define"
)

type SelsHandler struct {
	define.ServerInterface
}

func (h *SelsHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/sels/users", h.UserSels)
	rg.GET("/sels/tenants", h.TenantSels)
	rg.GET("/sels/projects", h.ProjectSels)
	rg.GET("/sels/environments", h.EnvironmentSels)
	rg.GET("/sels/applications", h.ApplicationSels)
}
