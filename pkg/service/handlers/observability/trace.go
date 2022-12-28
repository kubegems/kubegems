package observability

import (
	"context"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/observe"
	"kubegems.io/kubegems/pkg/utils/agents"
)

// GetTrace GetTrace by trace_id
// @Tags        Observability
// @Summary     GetTrace by trace_id
// @Description GetTrace by trace_id
// @Accept      json
// @Produce     json
// @Param       cluster  path     string                                      true "集群名"
// @Param       trace_id path     string                                      true "trace id"
// @Success     200      {object} handlers.ResponseStruct{Data=observe.Trace} "resp"
// @Router      /v1/observability/cluster/{cluster}/traces/{trace_id} [get]
// @Security    JWT
func (h *ObservabilityHandler) GetTrace(c *gin.Context) {
	// 前端传来的是UTC时间
	var trace *observe.Trace
	if err := h.Execute(c.Request.Context(), c.Param("cluster"), func(ctx context.Context, cli agents.Client) error {
		observecli := observe.NewClient(cli, h.GetDB().WithContext(ctx))
		var err error
		trace, err = observecli.GetTrace(ctx, c.Param("trace_id"))
		return err
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, trace)
}
