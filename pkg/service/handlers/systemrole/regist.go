package systemrolehandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/server/define"
)

// SystemRoleHandler generated handler
type SystemRoleHandler struct {
	define.ServerInterface
}

func (h *SystemRoleHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/systemrole", h.CheckIsSysADMIN, h.ListSystemRole)
	rg.POST("/systemrole", h.CheckIsSysADMIN, h.PostSystemRole)
	rg.GET("/systemrole/:systemrole_id", h.CheckIsSysADMIN, h.RetrieveSystemRole)
	rg.DELETE("/systemrole/:systemrole_id", h.CheckIsSysADMIN, h.DeleteSystemRole)
	rg.GET("/systemrole/:systemrole_id/user", h.CheckIsSysADMIN, h.ListSystemRoleUser)
	rg.PUT("/systemrole/:systemrole_id/user/:user_id", h.CheckIsSysADMIN, h.PutSystemRoleUser)
	rg.DELETE("/systemrole/:systemrole_id/user/:user_id", h.CheckIsSysADMIN, h.DeleteSystemRoleUser)
}
