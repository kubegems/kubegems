package base

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/redis"
)

// BaseHandler is the base handler for all handlers
type BaseHandler struct {
	agents      *agents.ClientSet
	redis       *redis.Client
	modelClient client.ModelClientIface
}

func NewBaseHandler(agents *agents.ClientSet, modelClient client.ModelClientIface, redis *redis.Client) BaseHandler {
	return BaseHandler{
		agents: agents,
		redis:  redis,
	}
}

func (h *BaseHandler) Agents() *agents.ClientSet {
	return h.agents
}

func (h *BaseHandler) Model() client.ModelClientIface {
	return h.modelClient
}

func (h *BaseHandler) Redis() *redis.Client {
	return h.redis
}

type OnClusterFunc func(ctx context.Context, cli agents.Client) (interface{}, error)

func (h BaseHandler) ClusterFunc(cluster string, fun OnClusterFunc) restful.RouteFunction {
	return func(req *restful.Request, resp *restful.Response) {
		ctx := req.Request.Context()
		cli, err := h.Agents().ClientOf(ctx, cluster)
		if err != nil {
			handlers.BadRequest(resp, err)
			return
		}
		data, err := fun(ctx, cli)
		if err != nil {
			handlers.BadRequest(resp, err)
			return
		}
		if data != nil {
			handlers.BadRequest(resp, err)
		}
		handlers.OK(resp, data)
	}
}

func (h BaseHandler) Execute(ctx context.Context, cluster string, fun func(ctx context.Context, cli agents.Client) error) error {
	cli, err := h.Agents().ClientOf(ctx, cluster)
	if err != nil {
		return err
	}
	return fun(ctx, cli)
}
