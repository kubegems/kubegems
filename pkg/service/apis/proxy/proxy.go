package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type Proxy struct {
	Proxy http.Handler
}

func NewProxy(target string) (*Proxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	return &Proxy{Proxy: httputil.NewSingleHostReverseProxy(u)}, nil
}

func (p *Proxy) Handle(ctx *gin.Context) {
	p.Proxy.ServeHTTP(ctx.Writer, ctx.Request)
}
