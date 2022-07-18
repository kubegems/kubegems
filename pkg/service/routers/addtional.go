package routers

import (
	"context"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/apis"
	"kubegems.io/kubegems/pkg/service/apis/proxy"
)

func (r *Router) AddRestAPI(ctx context.Context, deps apis.Dependencies) error {
	// proxy
	modelsproxy, err := proxy.NewModelsProxy(deps.Opts.Models.Addr)
	if err != nil {
		return err
	}
	modelsproxyfun := modelsproxy.Handler
	r.gin.Any("/v1/tenants/:tenant/projects/:project/environments/:environment/modeldeployments/*path", modelsproxyfun)
	r.gin.Any("/v1/tenants/:tenant/projects/:project/environments/:environment/modeldeployments", modelsproxyfun)
	r.gin.Any("/v1/sources/*path", modelsproxyfun)
	r.gin.Any("/v1/sources", modelsproxyfun)

	// api
	apis, err := apis.InitAPI(ctx, deps)
	if err != nil {
		return err
	}
	modelsfun := func(gin *gin.Context) {
		apis.ServeHTTP(gin.Writer, gin.Request)
	}
	r.gin.Any("/v1/docs.json", modelsfun)
	r.gin.Any("/v1/plugins", modelsfun)
	r.gin.Any("/v1/plugins/:name", modelsfun)
	return nil
}
