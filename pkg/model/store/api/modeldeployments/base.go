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

package modeldeployments

import (
	"context"
	"encoding/json"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AppRef struct {
	Tenant    string `json:"tenant,omitempty"`
	Project   string `json:"project,omitempty"`
	Env       string `json:"environment,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"` // namespace of the environment
	Username  string `json:"username,omitempty"`
}

func (r AppRef) Json() string {
	content, _ := json.Marshal(r)
	return string(content)
}

func (r *AppRef) FromJson(content string) {
	_ = json.Unmarshal([]byte(content), r)
}

type APPFunc func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error)

func (o *ModelDeploymentAPI) AppRefFunc(req *restful.Request, resp *restful.Response, fun APPFunc) {
	ref := AppRef{
		Tenant:  req.PathParameter("tenant"),
		Project: req.PathParameter("project"),
		Env:     req.PathParameter("environment"),
		Name:    req.PathParameter("name"),
	}
	ref.Username, _ = req.Attribute("username").(string)
	// check permission
	ctx := req.Request.Context()

	innerfunc := func() (interface{}, error) {
		env := &models.Environment{
			EnvironmentName: ref.Env,
			Project: &models.Project{
				ProjectName: ref.Project,
				Tenant: &models.Tenant{
					TenantName: ref.Tenant,
				},
			},
		}
		if err := o.Database.DB().Preload("Cluster").Take(env).Error; err != nil {
			return nil, err
		}
		clustername, namespace := env.Cluster.ClusterName, env.Namespace
		ref.Namespace = namespace

		cli, err := o.Clientset.ClientOf(ctx, clustername)
		if err != nil {
			return nil, err
		}
		return fun(ctx, cli, ref)
	}

	if data, err := innerfunc(); err != nil {
		response.BadRequest(resp, err.Error())
	} else {
		response.OK(resp, data)
	}
}
