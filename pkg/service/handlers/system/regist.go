package systemhandler

import (
	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/server/define"
)

type SystemHandler struct {
	define.ServerInterface
}

func (h *SystemHandler) RegistRouter(rg *gin.RouterGroup) {
}
