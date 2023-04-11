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
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	cserice "kubegems.io/configer/service"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"
)

func registPlugins(rg *gin.RouterGroup, basehandler base.BaseHandler) error {
	configPlugin, err := cserice.NewPlugin(&PluginInfoGetter{BaseHandler: basehandler}, basehandler.GetDB())
	if err != nil {
		return err
	}
	if err := configPlugin.InitDatabase(); err != nil {
		return err
	}
	configPlugin.Handler.RegistRouter(rg)
	return nil
}

type PluginInfoGetter struct {
	base.BaseHandler
}

func (p *PluginInfoGetter) ClusterNameOf(tenant, project, environment string) (clusterName string) {
	cluster := &models.Cluster{}
	sql := `select clusters.cluster_name as cluster_name, clusters.id as id from clusters
	left join environments ON environments.cluster_id = clusters.id
	left join projects on projects.id = environments.project_id
	left join tenants on tenants.id = projects.tenant_id
	where environment_name = @environment and project_name = @project and tenant_name = @tenant limit 1`
	if err := p.GetDB().Raw(sql, map[string]interface{}{"tenant": tenant, "project": project, "environment": environment}).Find(&cluster).Error; err != nil {
		return ""
	}
	return cluster.ClusterName
}

func (p *PluginInfoGetter) NacosInfoOf(clusterName string) (addr, username, password string, err error) {
	cli, err := p.GetAgents().ClientOf(context.Background(), clusterName)
	if err != nil {
		return "", "", "", err
	}
	u := cli.BaseAddr()
	h := u.String() + "/v1/service-proxy"
	return h, "nacos", "nacos", nil
}

func (p *PluginInfoGetter) RoundTripperOf(clusterName string) (rt http.RoundTripper) {
	cli, _ := p.GetAgents().ClientOf(context.Background(), clusterName)
	return RoundTripOf(cli)
}

func (p *PluginInfoGetter) Username(c *gin.Context) string {
	u, exist := p.ContextUserOperator.GetContextUser(c)
	if !exist {
		return ""
	}
	return u.GetUsername()
}

// 临时修复
func RoundTripOf(cli agents.Client) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		realPath := strings.TrimPrefix(req.URL.Path, cli.BaseAddr().Path)
		return cli.DoRawRequest(req.Context(), agents.Request{
			Method:  req.Method,
			Path:    realPath,
			Query:   req.URL.Query(),
			Headers: req.Header,
			Body:    req.Body,
		})
	})
}

type RoundTripperFunc func(req *http.Request) (*http.Response, error)

func (c RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return c(req)
}
