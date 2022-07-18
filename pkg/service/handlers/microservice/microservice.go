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

package microservice

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	microserviceoptions "kubegems.io/kubegems/pkg/service/handlers/microservice/options"
)

type MicroServiceHandler struct {
	vsh *VirtualSpaceHandler
	vdh *VirtualDomainHandler
	igh *IstioGatewayHandler
}

func NewMicroServiceHandler(si base.BaseHandler, options *microserviceoptions.MicroserviceOptions) *MicroServiceHandler {
	return &MicroServiceHandler{
		vsh: &VirtualSpaceHandler{BaseHandler: si, MicroserviceOptions: options},
		vdh: &VirtualDomainHandler{BaseHandler: si, MicroserviceOptions: options},
		igh: &IstioGatewayHandler{BaseHandler: si, MicroserviceOptions: options},
	}
}

func (h *MicroServiceHandler) RegistRouter(rg *gin.RouterGroup) {
	h.vdh.RegistRouter(rg)
	h.vsh.RegistRouter(rg)
	h.igh.RegistRouter(rg)
}
