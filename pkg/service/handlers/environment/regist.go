package environmenthandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type EnvironmentHandler struct {
	base.BaseHandler
}

func (h *EnvironmentHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/environment", h.CheckIsSysADMIN, h.ListEnvironment)
	rg.GET("/environment/:environment_id", h.CheckByEnvironmentID, h.RetrieveEnvironment)
	rg.PUT("/environment/:environment_id", h.CheckByEnvironmentID, h.PutEnvironment)
	rg.DELETE("/environment/:environment_id", h.CheckByEnvironmentID, h.DeleteEnvironment)

	rg.GET("/environment/:environment_id/user", h.CheckByEnvironmentID, h.ListEnvironmentUser)
	rg.GET("/environment/:environment_id/user/:user_id", h.CheckByEnvironmentID, h.RetrieveEnvironmentUser)
	rg.POST("/environment/:environment_id/user", h.CheckByEnvironmentID, h.PostEnvironmentUser)
	rg.PUT("/environment/:environment_id/user/:user_id", h.CheckByEnvironmentID, h.PutEnvironmentUser)
	rg.DELETE("/environment/:environment_id/user/:user_id", h.CheckByEnvironmentID, h.DeleteEnvironmentUser)

	rg.POST("/environment/:environment_id/action/networkisolate", h.CheckByEnvironmentID, h.EnvironmentSwitch)
	rg.GET("/environment/:environment_id/resources", h.CheckByEnvironmentID, h.GetEnvironmentResource)
}
