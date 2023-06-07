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
	"net/url"
	"strings"

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
	targetPath := c.Param("action")
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
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
	cli, err := h.GetAgents().ClientOf(c.Request.Context(), cluster)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	req := c.Request
	rp := cli.ReverseProxy(&url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s.%s:%s", service, namespace, port),
	})
	// k8s apiserver proxy will modify the html response, so we need to correct the path
	// https://github.com/kubernetes/apimachinery/blob/7ed5d2d91a598ca4d125acac5061f2a12721bbe8/pkg/util/proxy/transport.go#L124
	rp.Transport = &proxyutil.Transport{
		// detect which uri the request from, and set the prepend path
		PathPrepend: func() string {
			proxyRequestURI, _ := url.ParseRequestURI(req.Header.Get("X-Forwarded-Uri"))
			if proxyRequestURI != nil {
				return proxyRequestURI.Path
			}
			webuiPrefix := "/api" // our webui proxy path
			return webuiPrefix + strings.TrimSuffix(req.URL.Path, targetPath)
		}(),
		TrimPrefix:   cli.Config().Addr.Path, // trim the base path
		RoundTripper: rp.Transport,
	}
	req.URL.Path = targetPath
	rp.ServeHTTP(c.Writer, c.Request)
}
