package filters

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
)

type AuditMiddleware struct{}

func (a *AuditMiddleware) FilterFunc(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	operatoion := req.SelectedRoute().Operation()
	u := req.Attribute("user")

	user := u.(client.CommonUserIfe)
	resourceContext := req.PathParameters()
	fmt.Println(req.Request.RemoteAddr, user.GetUsername(), operatoion, resourceContext)

	chain.ProcessFilter(req, resp)

	fmt.Println("result", resp.StatusCode())

}
