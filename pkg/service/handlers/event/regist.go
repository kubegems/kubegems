package eventhandler

import (
	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/server/define"
)

type EventHandler struct {
	define.ServerInterface
}

func (h *EventHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/event/:cluster", h.Event)
}
