package filters

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
)

type PermMiddleware struct{}

func (p *PermMiddleware) FilterFunc(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	fmt.Println("TODO: before 权限中间件")
	fmt.Println(req.PathParameters())
	chain.ProcessFilter(req, resp)
	fmt.Println("TODO: after 权限中间件")
}
