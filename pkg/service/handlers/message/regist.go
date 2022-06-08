package messagehandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
)

type MessageHandler struct {
	base.BaseHandler
}

func (h *MessageHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/message", h.ListMessage)
	rg.PUT("/message/:message_id", h.ReadMessage)
}
