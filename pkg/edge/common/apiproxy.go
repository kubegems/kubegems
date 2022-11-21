package common

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/edge/tunnel"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

func (a *EdgeClusterAPI) Proxy(req *restful.Request, resp *restful.Response) {
	uid, path := req.PathParameter("uid"), "/"+req.PathParameter("path")
	// the default agent address
	ctx := req.Request.Context()
	edgecluster, err := a.Cluster.Get(ctx, uid)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	agentaddress := edgecluster.Status.Manufacture[AnnotationKeyEdgeAgentAddress]
	if agentaddress == "" {
		agentaddress = "http://127.0.0.1:8080" // fallback
	}
	log.Info("proxy http", "uid", uid, "target", agentaddress, "path", path)
	proxyTarget, err := url.Parse(agentaddress)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	proxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = proxyTarget.Scheme
			req.URL.Host = proxyTarget.Host
			req.URL.Path = path
		},
		Transport: tunnel.TransportOnTunnel(a.Tunnel, uid),
	}
	proxy.ServeHTTP(resp, req.Request)
}
