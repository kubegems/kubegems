package approveHandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/server/define"
)

type ApproveHandler struct {
	define.ServerInterface
}

func (h *ApproveHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/approve", h.ListApproves)
	rg.POST("/approve/:id/pass",
		h.CheckIsSysADMIN, h.Pass)
	rg.POST("/approve/:id/reject",
		h.CheckIsSysADMIN, h.Reject)
}
