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

	// models store
	r.gin.Any("/v1/docs.json", p.Handle)
	r.gin.Any("/v1/admin/*path", p.Handle)
	r.gin.Any("/v1/sources/*path", p.Handle)
	r.gin.Any("/v1/sources", p.Handle)
	return nil
}
