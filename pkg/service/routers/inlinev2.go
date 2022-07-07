package routers

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	v2 "kubegems.io/kubegems/pkg/service/v2"
)

func (r *Router) AddV2Restful(ctx context.Context, deps v2.Dependencies) error {
	apiv2, err := v2.InitAPI(ctx, deps)
	if err != nil {
		return err
	}
	r.gin.Any("/v2/*path", func(ctx *gin.Context) {
		ctx.Request.URL.Path = strings.TrimPrefix(ctx.Request.URL.Path, "/v2")
		ctx.Request.URL.RawPath = strings.TrimPrefix(ctx.Request.URL.RawPath, "/v2")
		apiv2.ServeHTTP(ctx.Writer, ctx.Request)
	})
	return nil
}
