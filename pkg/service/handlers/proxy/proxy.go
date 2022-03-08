package proxy

import (
	"net/http"
	"net/http/httputil"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/handlers/base"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/agents"
)

const (
	AgentModeApiServer = "apiServerProxy"
	AgentModeAHTTP     = "http"
	AgentModeHTTPS     = "https"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type ProxyHandler struct {
	base.BaseHandler
}

// 不需要swagger
func (h *ProxyHandler) Proxy(c *gin.Context) {
	// TODO: 可以根据schema判断
	if isstream, _ := strconv.ParseBool(c.Query("stream")); isstream {
		h.ProxyWebsocket(c)
	} else {
		h.ProxyHTTP(c)
	}
}

func (h *ProxyHandler) ProxyHTTP(c *gin.Context) {
	proxyPath := c.Param("action")
	cluster := c.Param("cluster")
	proxyobj := ParseProxyObj(c, proxyPath)

	// 审计
	h.AuditProxyFunc(c, proxyobj)

	// 权限
	if proxyobj.InNamespace() {
		h.CheckByClusterNamespace(c)
		if c.IsAborted() {
			return
		}
	}
	v, err := h.GetAgents().ClientOf(c.Request.Context(), cluster)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.ReverseProxyOn(v).ServeHTTP(c.Writer, c.Request)
}

func (h *ProxyHandler) ReverseProxyOn(cli agents.Client) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Path = getTargetPath(cli.Name(), req)
		},
		Transport: RoundTripOf(cli),
	}
}

// RoundTripOf
func RoundTripOf(cli agents.Client) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return cli.DoRawRequest(req.Context(), agents.Request{
			Method:  req.Method,
			Path:    req.URL.Path,
			Query:   req.URL.Query(),
			Headers: req.Header,
			Body:    req.Body,
		})
	})
}

type RoundTripperFunc func(req *http.Request) (*http.Response, error)

func (c RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return c(req)
}

func (h *ProxyHandler) ProxyWebsocket(c *gin.Context) {
	cluster := c.Param("cluster")
	proxyPath := c.Param("action")

	v, err := h.GetAgents().ClientOf(c.Request.Context(), cluster)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	proxyobj := ParseProxyObj(c, proxyPath)
	if proxyobj.InNamespace() {
		h.CheckByClusterNamespace(c)
		if c.IsAborted() {
			return
		}
	}

	headers := http.Header{}
	for key, values := range c.Request.URL.Query() {
		headers.Add(key, strings.Join(values, ","))
	}

	proxyConn, _, err := v.DialWebsocket(c.Request.Context(), proxyPath, headers)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	localConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	user, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, "请登录")
		return
	}
	var auditFunc func(string)
	env := h.GetCacheLayer().GetGlobalResourceTree().Tree.FindNodeByClusterNamespace(cluster, proxyobj.Namespace)
	if env != nil {
		log.Infof("proxy websocket, cluster is [%v], proxyobj is [%v]", cluster, proxyobj)
		parents := h.GetCacheLayer().GetGlobalResourceTree().Tree.FindParents(models.ResEnvironment, env.ID)
		auditFunc = h.WebsocketAuditFunc(user.GetUsername(), parents, c.ClientIP(), proxyobj)
	} else {
		log.Infof("proxy websocket can't find env, cluster is [%v], proxyobj is [%v]", cluster, proxyobj)
		auditFunc = h.WebsocketAuditFunc(user.GetUsername(), nil, c.ClientIP(), proxyobj)
	}
	Transport(localConn, proxyConn, c, user, auditFunc)
}

func getTargetPath(name string, req *http.Request) (realpath string) {
	prefix := path.Join("/v1/proxy/cluster", name)
	trimed := strings.TrimPrefix(req.URL.Path, prefix)
	if strings.HasPrefix(trimed, "/custom") {
		return trimed
	} else {
		return "/v1" + trimed
	}
}
