package microservice

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/server/define"
)

type MicroServiceHandler struct {
	vsh *VirtualSpaceHandler
	vdh *VirtualDomainHandler
	igh *IstioGatewayHandler
}

func NewMicroServiceHandler(si define.ServerInterface) *MicroServiceHandler {
	return &MicroServiceHandler{
		vsh: &VirtualSpaceHandler{ServerInterface: si, Agents: si.GetAgentsClientSet()},
		vdh: &VirtualDomainHandler{ServerInterface: si},
		igh: &IstioGatewayHandler{ServerInterface: si},
	}
}

func (h *MicroServiceHandler) RegistRouter(rg *gin.RouterGroup) {
	h.vdh.RegistRouter(rg)
	h.vsh.RegistRouter(rg)
	h.igh.RegistRouter(rg)
}
