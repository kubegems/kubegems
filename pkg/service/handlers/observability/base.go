package observability

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type ObservabilityHandler struct {
	base.BaseHandler
}

func (h *ObservabilityHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/:service", h.CheckByClusterNamespace, h.GetMonitorCollector)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/monitor", h.CheckByClusterNamespace, h.AddOrUpdateMonitorCollector)
	rg.DELETE("/observability/cluster/:cluster/namespaces/:namespace/monitor/:service", h.CheckByClusterNamespace, h.DeleteMonitorCollector)
}
