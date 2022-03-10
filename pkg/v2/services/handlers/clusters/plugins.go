package clusterhandler

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/v2/services/handlers"
)

func (h *Handler) ListPlugins(req *restful.Request, resp *restful.Response) {
	h.cluster(req, resp, func(ctx context.Context, cluster string, cli agents.Client) (interface{}, error) {
		return cli.Extend().ListPlugins(req.Request.Context())
	})
}

func (h *Handler) PluginSwitch(req *restful.Request, resp *restful.Response) {
	h.cluster(req, resp, func(ctx context.Context, cluster string, cli agents.Client) (interface{}, error) {
		pluginType := req.QueryParameter("type")
		pluginName := req.QueryParameter("plugin")
		if err := cli.Extend().EnablePlugin(req.Request.Context(), pluginType, pluginName); err != nil {
			return nil, err
		}
		return "", nil
	})
}

func (h *Handler) cluster(req *restful.Request, resp *restful.Response, fun func(ctx context.Context, cluster string, cli agents.Client) (interface{}, error)) {
	ctx := req.Request.Context()
	cluster, err := h.getCluster(ctx, req.PathParameter("cluster"))
	if err != nil {
		handlers.BadRequest(resp, err)
	}
	h.ClusterFunc(cluster.Name, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		return fun(ctx, cluster.Name, cli)
	})(req, resp)
}
