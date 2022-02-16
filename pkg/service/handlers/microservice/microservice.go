package microservice

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
	microserviceoptions "kubegems.io/pkg/service/handlers/microservice/options"
)

type MicroServiceHandler struct {
	vsh *VirtualSpaceHandler
	vdh *VirtualDomainHandler
	igh *IstioGatewayHandler
}

func NewMicroServiceHandler(si base.BaseHandler, options *microserviceoptions.MicroserviceOptions) *MicroServiceHandler {
	return &MicroServiceHandler{
		vsh: &VirtualSpaceHandler{BaseHandler: si, MicroserviceOptions: options},
		vdh: &VirtualDomainHandler{BaseHandler: si},
		igh: &IstioGatewayHandler{BaseHandler: si},
	}
}

func (h *MicroServiceHandler) RegistRouter(rg *gin.RouterGroup) {
	h.vdh.RegistRouter(rg)
	h.vsh.RegistRouter(rg)
	h.igh.RegistRouter(rg)
}
