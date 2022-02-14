package proxy

import (
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/server/define"
	"kubegems.io/pkg/service/handlers"
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
	define.ServerInterface
	clients *agents.ClientSet
}

func NewProxyHandler(server define.ServerInterface) *ProxyHandler {
	return &ProxyHandler{clients: server.GetAgentsClientSet(), ServerInterface: server}
}

// 不需要swagger
func (h *ProxyHandler) Proxy(c *gin.Context) {
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
	v, err := h.GetAgentsClientSet().ClientOf(c.Request.Context(), cluster)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	v.ProxyClient.HTTPProxy.ServeHTTP(c.Writer, c.Request)
}

func (h *ProxyHandler) ProxyWebsocket(c *gin.Context) {
	cluster := c.Param("cluster")
	proxyPath := c.Param("action")

	v, err := h.GetAgentsClientSet().ClientOf(c.Request.Context(), cluster)
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

	agentBaseAddr := v.BaseAddr

	var scheme string
	if agentBaseAddr.Scheme == "http" {
		scheme = "ws"
	} else {
		scheme = "wss"
	}
	headers := http.Header{}
	for key, values := range c.Request.URL.Query() {
		v := strings.Join(values, ",")
		headers.Add(key, v)
	}
	wsu := &url.URL{
		Scheme:   scheme,
		Host:     agentBaseAddr.Host,
		Path:     path.Join(agentBaseAddr.Path + proxyPath),
		RawQuery: c.Request.URL.Query().Encode(),
	}

	proxyConn, _, err := v.ProxyClient.WebsockerDialer.DialContext(c.Request.Context(), wsu.String(), headers)
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
		auditFunc = h.WebsocketAuditFunc(user.Username, parents, c.ClientIP(), proxyobj)
	} else {
		log.Infof("proxy websocket can't find env, cluster is [%v], proxyobj is [%v]", cluster, proxyobj)
		auditFunc = h.WebsocketAuditFunc(user.Username, nil, c.ClientIP(), proxyobj)
	}
	Transport(localConn, proxyConn, c, user, auditFunc)
}
