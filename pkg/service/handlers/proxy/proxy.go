// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"net/http"
	"net/http/httputil"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"
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

	// NOTICE:
	// why add query to headers? due the issue below, proxy params can't pass via api-server proxy
	// https://github.com/kubernetes/kubernetes/issues/89360
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
		handlers.Unauthorized(c, i18n.Errorf(c, "unauthorized, please login first"))
		return
	}
	var auditFunc func(string)
	env := h.ModelCache().FindEnvironment(cluster, proxyobj.Namespace)
	if env != nil {
		log.Infof("proxy websocket, cluster is [%v], proxyobj is [%v]", cluster, proxyobj)
		parents := h.ModelCache().FindParents(models.ResEnvironment, env.GetID())
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
