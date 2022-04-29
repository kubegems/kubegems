package alerthandler

import (
	"sync"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type AlertsHandler struct {
	base.BaseHandler
}

func (h *AlertsHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/alerts/cluster/:cluster/namespaces/:namespace/alert", h.CheckByClusterNamespace, h.ListAlertRule)
	rg.GET("/alerts/cluster/:cluster/namespaces/:namespace/alert/:name", h.CheckByClusterNamespace, h.GetAlertRule)
	rg.POST("/alerts/cluster/:cluster/namespaces/:namespace/alert/:name/actions/disable",
		h.CheckByClusterNamespace, h.DisableAlertRule)
	rg.POST("/alerts/cluster/:cluster/namespaces/:namespace/alert/:name/actions/enable",
		h.CheckByClusterNamespace, h.EnableAlertRule)
	rg.POST("/alerts/cluster/:cluster/namespaces/:namespace/alert",
		h.CheckByClusterNamespace, h.CreateAlertRule)
	rg.PUT("/alerts/cluster/:cluster/namespaces/:namespace/alert/:name",
		h.CheckByClusterNamespace, h.ModifyAlertRule)
	rg.DELETE("/alerts/cluster/:cluster/namespaces/:namespace/alert/:name",
		h.CheckByClusterNamespace, h.DeleteAlertRule)
	rg.GET("/alerts/cluster/:cluster/namespaces/:namespace/alert/:name/history",
		h.CheckByClusterNamespace, h.AlertHistory)
	rg.GET("/alerts/cluster/:cluster/namespaces/:namespace/alert/:name/repeats",
		h.CheckByClusterNamespace, h.AlertRepeats)

	// TODO: 权限
	rg.GET("/alerts/blacklist", h.ListBlackList)
	rg.POST("/alerts/blacklist", h.AddToBlackList)
	rg.DELETE("/alerts/blacklist/:fingerprint", h.RemoveInBlackList)

	rg.GET("/alerts/today", h.AlertToday)
	rg.GET("/alerts/graph", h.AlertGraph)
	rg.GET("/alerts/group", h.AlertByGroup)
	rg.GET("/alerts/tenant/:tenant_id/search", h.CheckByTenantID, h.SearchAlert)
}

type AlertmanagerConfigHandler struct {
	*AlertsHandler

	m sync.Mutex
}

func (h *AlertmanagerConfigHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/alerts/cluster/:cluster/namespaces/:namespace/receiver", h.CheckByClusterNamespace, h.ListReceiver)
	rg.POST("/alerts/cluster/:cluster/namespaces/:namespace/receiver",
		h.CheckByClusterNamespace, h.CreateReceiver)
	rg.PUT("/alerts/cluster/:cluster/namespaces/:namespace/receiver/:name",
		h.CheckByClusterNamespace, h.ModifyReceiver)
	rg.DELETE("/alerts/cluster/:cluster/namespaces/:namespace/receiver/:name",
		h.CheckByClusterNamespace, h.DeleteReceiver)
	rg.POST("/alerts/cluster/:cluster/namespaces/:namespace/receiver/:name/actions/test",
		h.CheckByClusterNamespace, h.TestEmail)
}
