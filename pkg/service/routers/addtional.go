package routers

import (
	"context"

	"kubegems.io/kubegems/pkg/service/apis"
)

func (r *Router) AddRestAPI(ctx context.Context, deps apis.Dependencies) error {
	apis, err := apis.NewModelsProxy(deps.Opts.Models.Addr)
	if err != nil {
		return err
	}
	modelsfun := apis.Handler
	r.gin.Any("/v1/docs.json", modelsfun)
	r.gin.Any("/v1/tenants/:tenant/projects/:project/environments/:environment/modeldeployments/*path", modelsfun)
	r.gin.Any("/v1/tenants/:tenant/projects/:project/environments/:environment/modeldeployments", modelsfun)
	r.gin.Any("/v1/sources/*path", modelsfun)
	r.gin.Any("/v1/sources", modelsfun)
	return nil
}
