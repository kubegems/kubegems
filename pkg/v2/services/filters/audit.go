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

package filters

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/v2/services/auth/user"
)

type AuditMiddleware struct{}

func (a *AuditMiddleware) FilterFunc(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	var op string
	route := req.SelectedRoute()
	if route != nil {
		op = route.Operation()
	}
	u := req.Attribute("user")
	var username string
	if u != nil {
		user := u.(user.CommonUserIface)
		username = user.GetUsername()
	}
	log.Info("in audit middleware", "remoteaddr", req.Request.RemoteAddr, "username", username, "opration", op)
	chain.ProcessFilter(req, resp)
}

func NewAuditMiddleware() *AuditMiddleware {
	return &AuditMiddleware{}
}
