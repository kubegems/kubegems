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

package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"kubegems.io/kubegems/pkg/apis/edge/common"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/library/rest/request"
	"kubegems.io/library/rest/response"
)

func (a *EdgeClusterAPI) Proxy(resp http.ResponseWriter, req *http.Request) {
	uid, path := request.Path(req, "uid", ""), "/"+request.Path(req, "path", "")
	// the default agent address
	ctx := req.Context()
	edgecluster, err := a.Cluster.ClusterStore.Get(ctx, uid)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	agentaddress := edgecluster.Status.Manufacture[common.AnnotationKeyEdgeAgentAddress]
	if agentaddress == "" {
		agentaddress = common.AnnotationValueDefaultEdgeAgentAddress // fallback
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
			backcompactAddV1Prefix(req)
		},
		Transport: a.Tunnel.TransportOnTunnel(uid),
	}
	proxy.ServeHTTP(resp, req)
}

// in original kubegems, api proxy to agent added "/v1" prefix
func backcompactAddV1Prefix(req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/v1") ||
		req.URL.Path == "/healthz" ||
		req.URL.Path == "/version" ||
		strings.HasPrefix(req.URL.Path, "/custom") ||
		strings.HasPrefix(req.URL.Path, "/internal") {
		return
	}
	// add "/v1" prefix
	req.URL.Path = "/v1" + req.URL.Path
}
