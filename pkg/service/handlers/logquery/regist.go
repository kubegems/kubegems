package logqueryhandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type LogQuerySnapshotHandler struct {
	base.BaseHandler
}

type LogQueryHistoryHandler struct {
	base.BaseHandler
}

func (h *LogQueryHistoryHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/logqueryhistory", h.ListLogQueryHistory)
	rg.POST("/logqueryhistory", h.PostLogQueryHistory)
	rg.DELETE("/logqueryhistory", h.BatchDeleteLogQueryHistory)
	rg.DELETE("/logqueryhistory/:logqueryhistory_id",
		h.DeleteLogQueryHistory)
}

func (h *LogQuerySnapshotHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/logquerysnapshot", h.ListLogQuerySnapshot)
	rg.GET("/logquerysnapshot/:logquerysnapshot_id", h.RetrieveLogQuerySnapshot)
	rg.DELETE("/logquerysnapshot/:logquerysnapshot_id",
		h.DeleteLogQuerySnapshot)
	rg.POST("/logquerysnapshot",
		h.PostLogQuerySnapshot)
}
