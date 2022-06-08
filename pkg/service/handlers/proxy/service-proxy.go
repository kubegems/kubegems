package proxy

import (
	"fmt"
	"path"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers"
	proxyutil "kubegems.io/kubegems/pkg/service/handlers/proxy/util"
	"kubegems.io/kubegems/pkg/utils/slice"
)

func (h *ProxyHandler) ProxyService(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	service := c.Param("service")
	port := c.Param("port")
	agentcli, err := h.GetAgents().ClientOf(c.Request.Context(), cluster)
	if err != nil {
		handlers.NotOK(c, err)
	}
	action := c.Param("action")

	agentPrefix := "/service-proxy"

	req := c.Copy()
	req.Request.Header.Set("namespace", namespace)
	req.Request.Header.Set("service", service)
	req.Request.Header.Set("port", port)
	if action == "" || action == "/" {
		req.Request.URL.Path = agentPrefix + "/_"
	} else {
		req.Request.URL.Path = path.Join(agentPrefix, action)
	}

	nswhiteList := []string{"istio-system", "observability"}
	svcwhiteList := []string{"kiali", "jaeger-query"}
	if !slice.ContainStr(nswhiteList, namespace) {
		handlers.Forbidden(c, "forbidden")
		return
	}
	if !slice.ContainStr(svcwhiteList, service) {
		handlers.Forbidden(c, "forbidden")
		return
	}

	reversep := h.ReverseProxyOn(agentcli)
	reversep.Transport = &proxyutil.Transport{
		PathPrepend:   fmt.Sprintf("/api/v1/service-proxy/cluster/%s/namespace/%s/service/%s/port/%s/", cluster, namespace, service, port),
		AgentPrefix:   agentPrefix,
		RoundTripper:  reversep.Transport,
		AgentBaseAddr: agentcli.BaseAddr().Path,
	}
	reversep.ServeHTTP(c.Writer, req.Request)
}
