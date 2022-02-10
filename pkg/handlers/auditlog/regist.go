package auditloghandler

import (
	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/server/define"
)

type AuditLogHandler struct {
	define.ServerInterface
}

func (h *AuditLogHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/auditlog", h.ListAuditLog)
	rg.GET("/auditlog/:auditlog_id", h.RetrieveAuditLog)
}
