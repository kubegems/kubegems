package eventhandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/lokilog"
)

type EventHandler struct {
	*lokilog.LogHandler
}

func (h *EventHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/event/:cluster", h.Event)
}
