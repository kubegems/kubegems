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
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/aaa/audit"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	"kubegems.io/kubegems/pkg/service/models"
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

	decision, err := h.checkPublic(c, proxyobj)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	switch decision {
	case Allow:
		break
	case Deny:
		handlers.Forbidden(c, fmt.Errorf("no permission to access %s", proxyobj.Path))
		return
	case NoOpinion:
		ns := proxyobj.GetNamespace()
		// nolint: nestif
		if ns == "" {
			ok, err := h.HasSystemAdminPerm(c)
			if err != nil {
				handlers.NotOK(c, err)
				return
			}
			if !ok {
				handlers.Forbidden(c, i18n.Errorf(c, "no permission to access cluster scope resources"))
				return
			}
		} else {
			// 权限
			ok, err := h.HasNamespacePerm(c, cluster, proxyobj.GetNamespace())
			if err != nil {
				handlers.NotOK(c, err)
				return
			}
			if !ok {
				handlers.Forbidden(c, i18n.Errorf(c, "don't have permission to access namespace %s", proxyobj.GetNamespace()))
				return
			}
		}
	}
	cli, err := h.GetAgents().ClientOf(c.Request.Context(), cluster)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if !strings.HasPrefix(proxyPath, "/custom") {
		proxyPath = path.Join("/v1", proxyPath)
	}
	c.Request.URL.Path = proxyPath
	cli.ReverseProxy().ServeHTTP(c.Writer, c.Request)
}

type AuthorizationDecision string

const (
	Allow     AuthorizationDecision = "allow"
	Deny      AuthorizationDecision = "deny"
	NoOpinion AuthorizationDecision = "noOpinion"
)

var PublicPath = map[string]func(h *ProxyHandler, c *gin.Context, obj *audit.ProxyObject) (AuthorizationDecision, error){
	"/plugins":                                               allowSimplePlugin,
	"/api-resources":                                         alwaysAllow,
	"/custom/prometheus/v1/matrix":                           alwaysAllow,
	"/custom/prometheus/v1/vector":                           alwaysAllow,
	"/storage.k8s.io/v1/storageclasses":                      alwaysAllow,
	"/networking.k8s.io/v1/ingressclasses":                   alwaysAllow,
	"/gems.kubegems.io/v1beta1/tenantgateways":               alwaysAllow,
	"/gems.kubegems.io/v1beta1/tenantnetworkpolicies":        checHasTenantPerm,
	"/gems.kubegems.io/v1beta1/tenantresourcequotas":         checHasTenantPerm,
	"/argoproj.io/v1alpha1/namespaces/kubegems/applications": allowUpdateArgoApp,
}

func allowSimplePlugin(h *ProxyHandler, c *gin.Context, obj *audit.ProxyObject) (AuthorizationDecision, error) {
	issimple := c.Query("simple")
	if issimple == "true" {
		return Allow, nil
	}
	return NoOpinion, nil
}

func allowUpdateArgoApp(h *ProxyHandler, c *gin.Context, obj *audit.ProxyObject) (AuthorizationDecision, error) {
	if obj.Method == http.MethodPatch {
		return Allow, nil
	}
	return NoOpinion, nil
}

func alwaysAllow(h *ProxyHandler, c *gin.Context, obj *audit.ProxyObject) (AuthorizationDecision, error) {
	return Allow, nil
}

func checHasTenantPerm(h *ProxyHandler, c *gin.Context, obj *audit.ProxyObject) (AuthorizationDecision, error) {
	ok, err := h.HasTenantPerm(c, obj.Name)
	if err != nil {
		return Deny, err
	}
	if !ok {
		return Deny, i18n.Errorf(c, "no permission to access tenant %s", obj.Name)
	}
	return Allow, nil
}

func (h *ProxyHandler) checkPublic(c *gin.Context, obj *audit.ProxyObject) (AuthorizationDecision, error) {
	for prefix, pub := range PublicPath {
		if strings.HasPrefix(obj.Path, prefix) {
			if pub == nil {
				return Allow, nil
			}
			return pub(h, c, obj)
		}
	}
	return NoOpinion, nil
}

func (h *ProxyHandler) ProxyWebsocket(c *gin.Context) {
	cluster := c.Param("cluster")
	proxyPath := c.Param("action")

	proxyobj := ParseProxyObj(c, proxyPath)

	ok, err := h.HasNamespacePerm(c, cluster, proxyobj.GetNamespace())
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if !ok {
		handlers.Forbidden(c, i18n.Errorf(c, "don't have permission to access namespace [%s]", proxyobj.GetNamespace()))
		return
	}

	// NOTICE:
	// why add query to headers? due the issue below, proxy params can't pass via api-server proxy
	// https://github.com/kubernetes/kubernetes/issues/89360
	headers := http.Header{}
	for key, values := range c.Request.URL.Query() {
		headers.Add(key, strings.Join(values, ","))
	}
	cli, err := h.GetAgents().ClientOf(c.Request.Context(), cluster)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	proxyConn, _, err := cli.Websocket().DialPath(c.Request.Context(), proxyPath, headers)
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
