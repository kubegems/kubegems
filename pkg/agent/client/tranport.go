// Copyright 2023 The kubegems.io Authors
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

package client

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/utils/route"
	"kubegems.io/library/net/httpproxy"
)

func (h *ClientTransport) Proxy(w http.ResponseWriter, r *http.Request) {
	h.proxyserver.ServeHTTP(w, r)
}

type ClientTransport struct {
	servepath   string
	proxyserver httpproxy.Server
}

func NewClientTransport() *ClientTransport {
	path := "/internal/proxy"
	return &ClientTransport{
		proxyserver: httpproxy.Server{Prefix: path},
		servepath:   path,
	}
}

func (h *ClientTransport) Register(r *route.Router) {
	r.GET(h.servepath, func(ctx *gin.Context) {
		h.Proxy(ctx.Writer, ctx.Request)
	})
	r.GET(h.servepath+"/{path}*", func(ctx *gin.Context) {
		h.Proxy(ctx.Writer, ctx.Request)
	})
}
