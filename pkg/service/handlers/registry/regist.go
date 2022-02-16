package registryhandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type RegistryHandler struct {
	base.BaseHandler
}

func (h *RegistryHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/registry", h.CheckIsSysADMIN, h.ListRegistry)
	rg.GET("/registry/:registry_id", h.CheckIsSysADMIN, h.RetrieveRegistry)
	rg.PUT("/registry/:registry_id",
		h.CheckIsSysADMIN, h.PutRegistry)
	rg.DELETE("/registry/:registry_id",
		h.CheckIsSysADMIN, h.DeleteRegistry)
}
