package lokiloghandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type LogHandler struct {
	base.BaseHandler
}

func (h *LogHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/log/:cluster_name/queryrange", h.QueryRange)
	rg.GET("/log/:cluster_name/labels", h.Labels)
	rg.GET("/log/:cluster_name/export", h.Export)
	rg.GET("/log/:cluster_name/label/:label/values", h.LabelValues)
	rg.GET("/log/:cluster_name/querylanguage", h.QueryLanguage)
	rg.GET("/log/:cluster_name/series", h.Series)
	rg.GET("/log/:cluster_name/context", h.Context)
}
