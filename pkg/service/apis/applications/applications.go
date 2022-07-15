package applications

import (
	"context"
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/service/handlers/application"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

type ApplicationsAPI struct {
	ApplicationProcessor *application.ApplicationProcessor
}

func NewApplicationsAPI(applicationProcessor *application.ApplicationProcessor) *ApplicationsAPI {
	return &ApplicationsAPI{ApplicationProcessor: applicationProcessor}
}

func (m *ApplicationsAPI) RegisterRoute(rg *route.Group) {
	rg.AddRoutes(
		route.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/images").To(
			m.DirectUpdateImage).Parameters(
			route.QueryParameter("image", "the image to update to"),
			route.QueryParameter("version", "the version of istio to update to").Optional(),
		),
	)
}

type NamedRefFunc func(ctx context.Context, ref application.PathRef) (interface{}, error)

func (m *ApplicationsAPI) NamedRefFunc(req *restful.Request, resp *restful.Response, fun NamedRefFunc) {
	ref := application.PathRef{
		Tenant:  req.PathParameter("tenant"),
		Project: req.PathParameter("project"),
		Env:     req.PathParameter("environment"),
		Name:    req.PathParameter("application"),
	}
	if data, err := fun(req.Request.Context(), ref); err != nil {
		response.Error(resp, err)
	} else {
		response.OK(resp, data)
	}
}

func (m *ApplicationsAPI) DirectUpdateImage(req *restful.Request, resp *restful.Response) {
	m.NamedRefFunc(req, resp, func(ctx context.Context, ref application.PathRef) (interface{}, error) {
		image := req.QueryParameter("image")
		if image == "" {
			return nil, fmt.Errorf("no image set,please set query item 'image'")
		}
		istioversion := req.QueryParameter("version")

		// update
		if err := m.ApplicationProcessor.UpdateImages(ctx, ref, []string{image}, istioversion); err != nil {
			return nil, err
		}
		// sync
		if err := m.ApplicationProcessor.Sync(ctx, ref); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}
