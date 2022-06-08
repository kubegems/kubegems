package registryhandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
)

type RegistryHandler struct {
	base.BaseHandler
}

func (h *RegistryHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/registry", h.CheckIsSysADMIN, h.ListRegistry)
	rg.GET("/registry/:registry_id", h.CheckIsSysADMIN, h.RetrieveRegistry)
	rg.PUT("/registry/:registry_id", h.CheckIsSysADMIN, h.PutRegistry)
	rg.DELETE("/registry/:registry_id", h.CheckIsSysADMIN, h.DeleteRegistry)

	rg.POST("/project/:project_id/registry", h.CheckByProjectID, h.PostProjectRegistry)
	rg.GET("/project/:project_id/registry", h.CheckByProjectID, h.ListProjectRegistry)
	rg.GET("/project/:project_id/registry/:registry_id", h.CheckByProjectID, h.RetrieveProjectRegistry)
	rg.PUT("/project/:project_id/registry/:registry_id", h.CheckByProjectID, h.PutProjectRegistry)
	rg.PATCH("/project/:project_id/registry/:registry_id", h.CheckByProjectID, h.SetDefaultProjectRegistry)
	rg.DELETE("/project/:project_id/registry/:registry_id", h.CheckByProjectID, h.DeleteProjectRegistry)
}
