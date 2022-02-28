package filters

import (
	"github.com/emicklei/go-restful/v3"
)

type PermMiddleware struct{}

func (p *PermMiddleware) FilterFunc(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	chain.ProcessFilter(req, resp)
}

func NewPermMiddleware() *PermMiddleware {
	return &PermMiddleware{}
}
