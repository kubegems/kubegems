package authsource

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type AuthSourceHandler struct {
	base.BaseHandler
}

func (h *AuthSourceHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/authsource", h.CheckIsSysADMIN, h.ListAuthSource)
	rg.POST("/authsource", h.CheckIsSysADMIN, h.Create)
	rg.PUT("/authsource/:source_id", h.CheckIsSysADMIN, h.Modify)
	rg.DELETE("/authsource/:source_id", h.CheckIsSysADMIN, h.Delete)
}
