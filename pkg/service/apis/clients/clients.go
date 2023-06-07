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

package clients

import (
	"net/http"

	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

type ClientsProxy struct {
	clients *agents.ClientSet
}

func NewClientsProxy(clients *agents.ClientSet) *ClientsProxy {
	return &ClientsProxy{
		clients: clients,
	}
}

// ProxyToCluster returns a http.Handler that proxies requests to the given cluster.
// Fixme: this is a temporary solution, we will use a better way to proxy requests in the future.
func (p *ClientsProxy) ProxyToCluster(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cli, err := p.clients.ClientOf(r.Context(), name)
		if err != nil {
			response.BadRequest(w, err.Error())
			return
		}
		cli.ReverseProxy().ServeHTTP(w, r)
	})
}
