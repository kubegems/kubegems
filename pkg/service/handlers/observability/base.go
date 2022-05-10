package observability

import (
	"sync"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type ObservabilityHandler struct {
	base.BaseHandler
	m sync.Mutex
}

func (h *ObservabilityHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor", h.CheckByClusterNamespace, h.GetMonitorCollector)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/monitor", h.CheckByClusterNamespace, h.AddOrUpdateMonitorCollector)
	rg.DELETE("/observability/cluster/:cluster/namespaces/:namespace/monitor", h.
		CheckByClusterNamespace, h.DeleteMonitorCollector)

	rg.PUT("/observability/cluster/:cluster/namespaces/:namespace/logging", h.CheckByClusterNamespace, h.NamespaceLogCollector)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/logging/apps", h.CheckByClusterNamespace, h.ListLogApps)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/logging/apps", h.CheckByClusterNamespace, h.AddAppLogCollector)

	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/logging/receivers", h.CheckByClusterNamespace, h.ListLoggingReceivers)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/logging/receivers", h.CheckByClusterNamespace, h.CreateLoggingReceiver)
	rg.PUT("/observability/cluster/:cluster/namespaces/:namespace/logging/receivers/:name", h.CheckByClusterNamespace, h.UpdateLoggingReceiver)
	rg.DELETE("/observability/cluster/:cluster/namespaces/:namespace/logging/receivers/:name", h.CheckByClusterNamespace, h.DeleteLoggingReceiver)

	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts", h.CheckByClusterNamespace, h.ListLoggingAlertRule)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts", h.CheckByClusterNamespace, h.CreateLoggingAlertRule)
}
