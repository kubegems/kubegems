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

package apis

import (
	"context"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/service/aaa/auth"
	"kubegems.io/kubegems/pkg/service/apis/oidc"
	"kubegems.io/kubegems/pkg/service/apis/plugins"
	"kubegems.io/kubegems/pkg/service/options"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/httputil/apiutil"
	"kubegems.io/kubegems/pkg/utils/redis"
)

type API struct{}

type Dependencies struct {
	Opts     *options.Options
	Agents   *agents.ClientSet
	Database *database.Database
	Gitp     *git.SimpleLocalProvider
	Argo     *argo.Client
	Redis    *redis.Client
}

func InitAPI(ctx context.Context, deps Dependencies) (http.Handler, error) {
	pluginsapi, err := plugins.NewPluginsAPI(deps.Agents)
	if err != nil {
		return nil, err
	}
	op, err := oidc.NewProvider(ctx, &oidc.OIDCOptions{
		Issuer:   deps.Opts.JWT.IssuerAddr,
		CertFile: deps.Opts.JWT.Cert,
		KeyFile:  deps.Opts.JWT.Key,
	})
	if err != nil {
		return nil, err
	}
	modules := []apiutil.RestModule{
		pluginsapi,
		op,
	}
	middlewares := []restful.FilterFunction{
		SkipIf(
			[]string{
				"/keys",
				"/.well-known/openid-configuration",
			},
			auth.NewAuthMiddleware(deps.Opts.JWT, nil).GoRestfulMiddleware, // authc
		),
	}
	return apiutil.NewRestfulAPI("", middlewares, modules), nil
}

func SkipIf(list []string, fun restful.FilterFunction) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		for _, v := range list {
			if strings.HasPrefix(req.Request.URL.Path, v) {
				chain.ProcessFilter(req, resp)
				return
			}
		}
		fun(req, resp, chain)
	}
}
