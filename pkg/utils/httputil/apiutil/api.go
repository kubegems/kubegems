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

package apiutil

import (
	"net/http"
	"path"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-logr/logr"
	"github.com/go-openapi/spec"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
	"kubegems.io/kubegems/pkg/version"
)

type RestModule interface {
	RegisterRoute(r *route.Group)
}

func NewRestfulAPI(prefix string, filters []restful.FilterFunction, modules []RestModule) http.Handler {
	return NewRestfulAPIWithHealthCheck(prefix, filters, modules, nil)
}

func NewRestfulAPIWithHealthCheck(prefix string, filters []restful.FilterFunction, modules []RestModule, healthCheckFun func() error) http.Handler {
	ws := new(restful.WebService)
	for _, filter := range filters {
		ws.Filter(filter)
	}

	root := route.NewGroup("")
	root.AddRoutes(
		route.GET("healthz").Tag("default").Doc("health check").To(route.Healthz(healthCheckFun)),
	)

	rg := route.NewGroup(prefix)
	for _, module := range modules {
		module.RegisterRoute(rg)
	}
	root.AddSubGroup(rg)

	(&route.Tree{RouteUpdateFunc: listWrrapperFunc, Group: root}).AddToWebService(ws)
	cros := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{".*"},
		AllowedMethods: []string{"*"},
	}
	// cross filter must set on webservice
	ws.Filter(cros.Filter)

	c := restful.NewContainer()
	c.Filter(LogFilter)
	c.Filter(c.OPTIONSFilter)
	c.ServiceErrorHandler(errhandlerfunc)

	apidocs := route.BuildOpenAPIWebService([]*restful.WebService{ws}, path.Join(prefix, "docs.json"), completeInfo)
	return c.Add(ws).Add(apidocs)
}

func errhandlerfunc(err restful.ServiceError, req *restful.Request, resp *restful.Response) {
	if req.Request.Method == http.MethodOptions {
		return
	}
	for header, values := range err.Header {
		for _, value := range values {
			resp.Header().Add(header, value)
		}
	}
	resp.WriteErrorString(err.Code, err.Message)
}

func LogFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	log := log.LogrLogger
	req.Request = req.Request.WithContext(logr.NewContext(req.Request.Context(), log))

	start := time.Now()
	chain.ProcessFilter(req, resp)
	duration := time.Since(start)
	log.Info(req.Request.URL.String(), "method", req.Request.Method, "code", resp.StatusCode(), "remote", req.Request.RemoteAddr, "duration", duration.String())
}

func completeInfo(s *spec.Swagger) {
	s.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "KubeGems",
			Description: "kubegems api",
			Contact: &spec.ContactInfo{
				ContactInfoProps: spec.ContactInfoProps{
					Name:  "kubegems",
					URL:   "http://kubegems.io",
					Email: "support@kubegems.io",
				},
			},
			Version: version.Get().GitVersion,
		},
	}
	s.Schemes = []string{"http", "https"}
	s.SecurityDefinitions = map[string]*spec.SecurityScheme{
		"jwt": spec.APIKeyAuth("Authorization", "header"),
	}
	s.Security = []map[string][]string{{"jwt": {}}}
}

func listWrrapperFunc(r *route.Route) {
	paged := false
	for _, item := range r.Params {
		if item.Kind == route.ParamKindQuery && item.Name == "page" {
			paged = true
			break
		}
	}
	for i, v := range r.Responses {
		//  if query parameters exist, response as a paged response
		if paged {
			r.Responses[i].Body = response.Response{Data: response.Page{List: v.Body}}
		} else {
			r.Responses[i].Body = response.Response{Data: v.Body}
		}
	}
}
