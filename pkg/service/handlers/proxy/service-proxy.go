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
	"path"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/i18n"
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
		handlers.Forbidden(c, i18n.Errorf(c, "forbidden"))
		return
	}
	if !slice.ContainStr(svcwhiteList, service) {
		handlers.Forbidden(c, i18n.Errorf(c, "forbidden"))
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
