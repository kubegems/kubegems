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

package routers

import (
	"context"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/apis"
	"kubegems.io/kubegems/pkg/service/apis/clients"
	"kubegems.io/kubegems/pkg/service/apis/proxy"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

func (r *Router) AddRestAPI(ctx context.Context, deps apis.Dependencies) error {
	apis, err := apis.InitAPI(ctx, deps)
	if err != nil {
		return err
	}
	apifun := func(gin *gin.Context) {
		apis.ServeHTTP(gin.Writer, gin.Request)
	}
	r.gin.Any("/v1/plugins", apifun)
	r.gin.Any("/.well-known/openid-configuration", apifun) // oidc discovery
	r.gin.Any("/keys", apifun)                             // oidc keys

	// just hardcode the path for now
	p, err := proxy.NewProxy(deps.Opts.Models.Addr)
	if err != nil {
		return err
	}
	// models store
	r.gin.Any("/v1/tenants/:tenant/projects/:project/environments/:environment/modeldeployments/*path", p.Handle)
	r.gin.Any("/v1/tenants/:tenant/projects/:project/environments/:environment/modeldeployments", p.Handle)
	r.gin.Any("/v1/docs.json", p.Handle)
	r.gin.Any("/v1/admin/*path", p.Handle)
	r.gin.Any("/v1/sources/*path", p.Handle)
	r.gin.Any("/v1/sources", p.Handle)

	// kubegems edge
	// just hardcode the path for now
	edgep, err := proxy.NewProxy(deps.Opts.Edge.Addr)
	if err != nil {
		return err
	}
	r.gin.Any("/v1/edge-clusters", edgep.Handle)
	r.gin.Any("/v1/edge-clusters/*path", edgep.Handle)
	r.gin.Any("/v1/edge-hubs", edgep.Handle)
	r.gin.Any("/v1/edge-hubs/*path", edgep.Handle)

	// agents proxy (internal)
	clientsproxy := clients.NewClientsProxy(deps.Agents)
	r.gin.Any("/internal/clusters", func(ctx *gin.Context) {
		clusters := deps.Agents.Clusters()
		response.OK(ctx.Writer, clusters)
	})
	r.gin.Any("/internal/agents/:name/*path", func(ctx *gin.Context) {
		name := ctx.Param("name")
		ctx.Request.URL.Path = ctx.Param("path") + "?" + ctx.Request.URL.RawQuery
		ctx.Request.URL.RawPath = ctx.Param("path") + "?" + ctx.Request.URL.RawQuery
		clientsproxy.HandlerToCluster(name).ServeHTTP(ctx.Writer, ctx.Request)
	})
	return nil
}
