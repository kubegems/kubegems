// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clusterhandler

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
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
