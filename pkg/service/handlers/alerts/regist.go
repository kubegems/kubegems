package alerthandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type AlertsHandler struct {
	base.BaseHandler
}

func (h *AlertsHandler) RegistRouter(rg *gin.RouterGroup) {
	// TODO: 权限
	rg.GET("/alerts/blacklist", h.ListBlackList)
	rg.POST("/alerts/blacklist", h.AddToBlackList)
	rg.DELETE("/alerts/blacklist/:fingerprint", h.RemoveInBlackList)

}
