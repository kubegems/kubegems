package auditloghandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type AuditLogHandler struct {
	base.BaseHandler
}

func (h *AuditLogHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/auditlog", h.ListAuditLog)
	rg.GET("/auditlog/:auditlog_id", h.RetrieveAuditLog)
}
