package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type ModelsProxy struct {
	Proxy http.Handler
}

func NewModelsProxy(target string) (*ModelsProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	return &ModelsProxy{Proxy: httputil.NewSingleHostReverseProxy(u)}, nil
}

func (p *ModelsProxy) Handler(ctx *gin.Context) {
	p.Proxy.ServeHTTP(ctx.Writer, ctx.Request)
}
