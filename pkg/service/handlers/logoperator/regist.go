package logoperatorhandler

import (
	"kubegems.io/pkg/service/handlers/base"
	"github.com/gin-gonic/gin"
)

type LogOperatorHandler struct {
	base.BaseHandler
}

func (h *LogOperatorHandler) RegistRouter(rg *gin.RouterGroup) {
	// TODO: uncomment graph & nodeagent uri ,it's tested
	// rg.GET("/logging/cluster/:cluster/namespaces/:namespace/flows/:flowid/graph", h.Graph)
	// rg.GET("/logging/cluster/:cluster/namespaces/:namespace/flows/:flowid/metircs", h.Metrics)
	rg.GET("/logging/cluster/:cluster/tenant/:tenant_id/flows", h.Flows)
	rg.GET("/logging/cluster/:cluster/tenant/:tenant_id/outputs", h.Outputs)
	// rg.POST("/logging/cluster/:cluster/namespaces/:namespace/nodeagent/:name", h.CreateNodeAgentLogCollector)
}
