package filters

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
)

type AuditMiddleware struct{}

func (a *AuditMiddleware) FilterFunc(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	fmt.Println("TODO: before 审计中间件")
	chain.ProcessFilter(req, resp)
	fmt.Println("TODO: after 审计中间件")
}
