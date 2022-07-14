package apiutil

import (
	"net/http"
	"path"
	"time"

	"github.com/emicklei/go-restful/v3"
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
	ws := new(restful.WebService)
	for _, filter := range filters {
		ws.Filter(filter)
	}

	rg := route.NewGroup(prefix)
	for _, module := range modules {
		module.RegisterRoute(rg)
	}

	(&route.Tree{RouteUpdateFunc: listWrrapperFunc, Group: rg}).AddToWebService(ws)
	ws.Filter(restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{".*"},
		AllowedMethods: []string{"*"},
	}.Filter)
	ws.Filter(LogFilter)
	ws.Filter(restful.OPTIONSFilter())

	healthz := new(restful.WebService)
	healthz.Path("healthz").Route(
		healthz.GET("").To(func(req *restful.Request, resp *restful.Response) {}).Doc("health check").Produces("text/plain").Writes("OK"),
	)
	return restful.DefaultContainer.
		Add(ws).
		Add(healthz).
		Add(route.BuildOpenAPIWebService([]*restful.WebService{ws}, path.Join(prefix, "docs.json"), completeInfo))
}

func LogFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
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
