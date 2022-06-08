package approveHandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
)

type ApproveHandler struct {
	base.BaseHandler
}

func (h *ApproveHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/approve", h.ListApproves)
	rg.POST("/approve/:id/pass",
		h.CheckIsSysADMIN, h.Pass)
	rg.POST("/approve/:id/reject",
		h.CheckIsSysADMIN, h.Reject)
}
