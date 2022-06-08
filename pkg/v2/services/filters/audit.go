package filters

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/v2/services/auth/user"
)

type AuditMiddleware struct{}

func (a *AuditMiddleware) FilterFunc(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	var op string
	route := req.SelectedRoute()
	if route != nil {
		op = route.Operation()
	}
	u := req.Attribute("user")
	var username string
	if u != nil {
		user := u.(user.CommonUserIface)
		username = user.GetUsername()
	}
	log.Info("in audit middleware", "remoteaddr", req.Request.RemoteAddr, "username", username, "opration", op)
	chain.ProcessFilter(req, resp)
}

func NewAuditMiddleware() *AuditMiddleware {
	return &AuditMiddleware{}
}
