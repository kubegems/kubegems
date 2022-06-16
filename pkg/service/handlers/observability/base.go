package observability

import (
	"sync"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	"kubegems.io/kubegems/pkg/utils/helm"
)

type ObservabilityHandler struct {
	base.BaseHandler
	AppStoreOpt       *helm.Options
	ChartmuseumClient *helm.ChartmuseumClient
	m                 sync.Mutex
}

func (h *ObservabilityHandler) RegistRouter(rg *gin.RouterGroup) {
	h.ChartmuseumClient = helm.MustNewChartMuseumClient(&helm.RepositoryConfig{URL: h.AppStoreOpt.Addr})
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor", h.CheckByClusterNamespace, h.GetMonitorCollector)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/monitor", h.CheckByClusterNamespace, h.AddOrUpdateMonitorCollector)
	rg.DELETE("/observability/cluster/:cluster/namespaces/:namespace/monitor", h.CheckByClusterNamespace, h.DeleteMonitorCollector)

	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/alerts", h.CheckByClusterNamespace, h.ListMonitorAlertRule)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/alerts/:name", h.CheckByClusterNamespace, h.GetMonitorAlertRule)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/monitor/alerts", h.CheckByClusterNamespace, h.CreateMonitorAlertRule)
	rg.PUT("/observability/cluster/:cluster/namespaces/:namespace/monitor/alerts/:name", h.CheckByClusterNamespace, h.UpdateMonitorAlertRule)
	rg.DELETE("/observability/cluster/:cluster/namespaces/:namespace/monitor/alerts/:name", h.CheckByClusterNamespace, h.DeleteMonitorAlertRule)

	rg.PUT("/observability/cluster/:cluster/namespaces/:namespace/logging", h.CheckByClusterNamespace, h.NamespaceLogCollector)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/logging/apps", h.CheckByClusterNamespace, h.ListLogApps)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/logging/apps", h.CheckByClusterNamespace, h.AddAppLogCollector)

	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/receivers", h.CheckByClusterNamespace, h.ListReceiver)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/receivers", h.CheckByClusterNamespace, h.CreateReceiver)
	rg.PUT("/observability/cluster/:cluster/namespaces/:namespace/receivers/:name", h.CheckByClusterNamespace, h.UpdateReceiver)
	rg.DELETE("/observability/cluster/:cluster/namespaces/:namespace/receivers/:name", h.CheckByClusterNamespace, h.DeleteReceiver)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/receivers/actions/test", h.CheckByClusterNamespace, h.TestEmail)

	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts", h.CheckByClusterNamespace, h.ListLoggingAlertRule)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts/:name", h.CheckByClusterNamespace, h.GetLoggingAlertRule)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts", h.CheckByClusterNamespace, h.CreateLoggingAlertRule)
	rg.PUT("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts/:name", h.CheckByClusterNamespace, h.UpdateLoggingAlertRule)
	rg.DELETE("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts/:name", h.CheckByClusterNamespace, h.DeleteLoggingAlertRule)

	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/alerts/:name/actions/enable", h.CheckByClusterNamespace, h.EnableAlertRule)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/alerts/:name/actions/disable", h.CheckByClusterNamespace, h.DisableAlertRule)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/alerts/:name/history", h.CheckByClusterNamespace, h.AlertHistory)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/alerts/:name/repeats", h.CheckByClusterNamespace, h.AlertRepeats)

	rg.GET("/observability/tenant/:tenant_id/alerts/today", h.CheckByClusterNamespace, h.AlertToday)
	rg.GET("/observability/tenant/:tenant_id/alerts/graph", h.CheckByClusterNamespace, h.AlertGraph)
	rg.GET("/observability/tenant/:tenant_id/alerts/group", h.CheckByClusterNamespace, h.AlertByGroup)
	rg.GET("/observability/tenant/:tenant_id/alerts/search", h.CheckByClusterNamespace, h.SearchAlert)

	// metrics
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/metrics/queryrange", h.CheckByClusterNamespace, h.QueryRange)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/metrics/labelvalues", h.CheckByClusterNamespace, h.LabelValues)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/metrics/labelnames", h.CheckByClusterNamespace, h.LabelNames)

	// template
	rg.GET("/observability/template/resources/:resource_name/rules/:rule_name", h.CheckIsSysADMIN, h.GetMetricTemplate)
	rg.POST("/observability/template/resources/:resource_name/rules/:rule_name", h.CheckIsSysADMIN, h.AddOrUpdateMetricTemplate)
	rg.DELETE("/observability/template/resources/:resource_name/rules/:rule_name", h.CheckIsSysADMIN, h.DeleteMetricTemplate)

	// dashboard
	rg.GET("/observability/environment/:environment_id/monitor/dashboard", h.CheckByEnvironmentID, h.ListDashboard)
	rg.GET("/observability/environment/:environment_id/monitor/dashboard/:dashboard_id", h.CheckByEnvironmentID, h.DashboardDetail)
	rg.POST("/observability/environment/:environment_id/monitor/dashboard", h.CheckByEnvironmentID, h.CreateDashboard)
	rg.PUT("/observability/environment/:environment_id/monitor/dashboard/:dashboard_id", h.CheckByEnvironmentID, h.UpdateDashboard)
	rg.DELETE("/observability/environment/:environment_id/monitor/dashboard/:dashboard_id", h.CheckByEnvironmentID, h.DeleteDashboard)

	// exporter
	rg.GET("/observability/monitor/exporters/:name/schema", h.ExporterSchema)
}
