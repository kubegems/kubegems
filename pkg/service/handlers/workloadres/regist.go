package workloadreshandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type WorkloadHandler struct {
	base.BaseHandler
}

func (h *WorkloadHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/resources/workload", h.ListWorkload)
	rg.DELETE("/resources/workload/:workload_id", h.DeleteWorkload)
}
