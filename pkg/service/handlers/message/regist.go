package messagehandler

import (
	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/server/define"
)

type MessageHandler struct {
	define.ServerInterface
}

func (h *MessageHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/message", h.ListMessage)
	rg.PUT("/message/:message_id", h.ReadMessage)
}
