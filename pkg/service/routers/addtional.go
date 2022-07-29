package routers

import (
	"context"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/apis"
)

func (r *Router) AddRestAPI(ctx context.Context, deps apis.Dependencies) error {
	apis, err := apis.InitAPI(ctx, deps)
	if err != nil {
		return err
	}
	modelsfun := func(gin *gin.Context) {
		apis.ServeHTTP(gin.Writer, gin.Request)
	}
	r.gin.Any("/v1/docs.json", modelsfun)
	r.gin.Any("/v1/plugins", modelsfun)
	r.gin.Any("/v1/admin/*path", modelsfun)
	r.gin.Any("/v1/sources/*path", modelsfun)
	r.gin.Any("/v1/sources", modelsfun)
	return nil
}
