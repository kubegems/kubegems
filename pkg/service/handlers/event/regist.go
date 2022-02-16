package eventhandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type EventHandler struct {
	base.BaseHandler
}

func (h *EventHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/event/:cluster", h.Event)
}
