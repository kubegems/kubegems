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
	"kubegems.io/kubegems/pkg/service/apis/proxy"
)

func (r *Router) AddRestAPI(ctx context.Context, deps apis.Dependencies) error {
	apis, err := apis.InitAPI(ctx, deps)
	if err != nil {
		return err
	}
	modelsfun := func(gin *gin.Context) {
		apis.ServeHTTP(gin.Writer, gin.Request)
	}

	// just hardcode the path for now
	p, err := proxy.NewProxy(deps.Opts.Models.Addr)
	if err != nil {
		return err
	}

	r.gin.Any("/v1/plugins", modelsfun)

	modeldeploymenyfuns := modelsfun
	if true { // modeldeployment proxy to the models
		modeldeploymenyfuns = p.Handle
	}
	r.gin.Any("/v1/tenants/:tenant/projects/:project/environments/:environment/modeldeployments/*path", modeldeploymenyfuns)
	r.gin.Any("/v1/tenants/:tenant/projects/:project/environments/:environment/modeldeployments", modeldeploymenyfuns)

	// models store
	r.gin.Any("/v1/docs.json", p.Handle)
	r.gin.Any("/v1/admin/*path", p.Handle)
	r.gin.Any("/v1/sources/*path", p.Handle)
	r.gin.Any("/v1/sources", p.Handle)
	return nil
}
