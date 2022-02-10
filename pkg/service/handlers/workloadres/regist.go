package workloadreshandler

import (
	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/server/define"
)

type WorkloadHandler struct {
	define.ServerInterface
}

func (h *WorkloadHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/resources/workload", h.ListWorkload)
	rg.DELETE("/resources/workload/:workload_id", h.DeleteWorkload)
}
