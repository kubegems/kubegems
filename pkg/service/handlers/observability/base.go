// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/status", h.CheckByClusterNamespace, h.MonitorCollectorStatus)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/monitor", h.CheckByClusterNamespace, h.AddOrUpdateMonitorCollector)
	rg.DELETE("/observability/cluster/:cluster/namespaces/:namespace/monitor", h.CheckByClusterNamespace, h.DeleteMonitorCollector)

	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/alerts", h.CheckByClusterNamespace, h.ListMonitorAlertRule)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/alerts/_/status", h.CheckByClusterNamespace, h.ListMonitorAlertRulesStatus)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/alerts/:name", h.CheckByClusterNamespace, h.GetMonitorAlertRule)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/monitor/alerts", h.CheckByClusterNamespace, h.CreateMonitorAlertRule)
	rg.PUT("/observability/cluster/:cluster/namespaces/:namespace/monitor/alerts/:name", h.CheckByClusterNamespace, h.UpdateMonitorAlertRule)
	rg.DELETE("/observability/cluster/:cluster/namespaces/:namespace/monitor/alerts/:name", h.CheckByClusterNamespace, h.DeleteMonitorAlertRule)

	rg.PUT("/observability/cluster/:cluster/namespaces/:namespace/logging", h.CheckByClusterNamespace, h.NamespaceLogCollector)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/logging/apps", h.CheckByClusterNamespace, h.ListLogApps)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/logging/apps", h.CheckByClusterNamespace, h.AddAppLogCollector)

	rg.GET("/observability/tenant/:tenant_id/channels", h.CheckByTenantID, h.ListChannels)
	rg.GET("/observability/tenant/:tenant_id/channels/:channel_id", h.CheckByTenantID, h.GetChannel)
	rg.POST("/observability/tenant/:tenant_id/channels", h.CheckByTenantID, h.CreateChannel)
	rg.PUT("/observability/tenant/:tenant_id/channels/:channel_id", h.CheckByTenantID, h.UpdateChannel)
	rg.DELETE("/observability/tenant/:tenant_id/channels/:channel_id", h.CheckByTenantID, h.DeleteChannel)
	rg.POST("/observability/tenant/:tenant_id/channels/:channel_id/test", h.TestChannel)

	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts", h.CheckByClusterNamespace, h.ListLoggingAlertRule)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts/_/status", h.CheckByClusterNamespace, h.ListLoggingAlertRulesStatus)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts/:name", h.CheckByClusterNamespace, h.GetLoggingAlertRule)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts", h.CheckByClusterNamespace, h.CreateLoggingAlertRule)
	rg.PUT("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts/:name", h.CheckByClusterNamespace, h.UpdateLoggingAlertRule)
	rg.DELETE("/observability/cluster/:cluster/namespaces/:namespace/logging/alerts/:name", h.CheckByClusterNamespace, h.DeleteLoggingAlertRule)

	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/alerts/:name/actions/enable", h.CheckByClusterNamespace, h.EnableAlertRule)
	rg.POST("/observability/cluster/:cluster/namespaces/:namespace/alerts/:name/actions/disable", h.CheckByClusterNamespace, h.DisableAlertRule)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/alerts/:name/history", h.CheckByClusterNamespace, h.AlertHistory)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/alerts/:name/repeats", h.CheckByClusterNamespace, h.AlertRepeats)

	rg.GET("/observability/tenant/:tenant_id/alerts/today", h.CheckByTenantID, h.AlertToday)
	rg.GET("/observability/tenant/:tenant_id/alerts/graph", h.CheckByTenantID, h.AlertGraph)
	rg.GET("/observability/tenant/:tenant_id/alerts/group", h.CheckByTenantID, h.AlertByGroup)
	rg.GET("/observability/tenant/:tenant_id/alerts/search", h.CheckByTenantID, h.SearchAlert)

	// metrics
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/metrics/queryrange", h.CheckByClusterNamespace, h.QueryRange)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/metrics/labelvalues", h.CheckByClusterNamespace, h.LabelValues)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/monitor/metrics/labelnames", h.CheckByClusterNamespace, h.LabelNames)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/otel/metrics/graphs", h.CheckByClusterNamespace, h.OtelMetricsGraphs)

	// template
	rg.GET("/observability/tenant/:tenant_id/template/scopes", h.CheckByTenantID, h.ListScopes)
	rg.GET("/observability/tenant/:tenant_id/template/scopes/:scope_id/resources", h.CheckByTenantID, h.ListResources)
	rg.GET("/observability/tenant/:tenant_id/template/resources/:resource_id/rules", h.CheckByTenantID, h.ListRules)
	rg.GET("/observability/tenant/:tenant_id/template/rules/:rule_id", h.CheckByTenantID, h.GetRule)
	rg.POST("/observability/tenant/:tenant_id/template/rules", h.CheckByTenantID, h.AddRules)
	rg.PUT("/observability/tenant/:tenant_id/template/rules/:rule_id", h.CheckByTenantID, h.UpdateRules)
	rg.DELETE("/observability/tenant/:tenant_id/template/rules/:rule_id", h.CheckByTenantID, h.DeleteRules)
	rg.GET("/observability/tenant/:tenant_id/template/search", h.CheckByTenantID, h.SearchTpl)

	// dashboard
	rg.GET("/observability/environment/:environment_id/monitor/dashboard", h.CheckByEnvironmentID, h.ListDashboard)
	rg.GET("/observability/environment/:environment_id/monitor/dashboard/:dashboard_id", h.CheckByEnvironmentID, h.DashboardDetail)
	rg.POST("/observability/environment/:environment_id/monitor/dashboard", h.CheckByEnvironmentID, h.CreateDashboard)
	rg.PUT("/observability/environment/:environment_id/monitor/dashboard/:dashboard_id", h.CheckByEnvironmentID, h.UpdateDashboard)
	rg.DELETE("/observability/environment/:environment_id/monitor/dashboard/:dashboard_id", h.CheckByEnvironmentID, h.DeleteDashboard)
	rg.GET("/observability/environment/:environment_id/monitor/dashboard/:dashboard_id/query", h.CheckByEnvironmentID, h.DashboardQuery)

	rg.GET("/observability/template/dashboard", h.ListDashboardTemplates)
	rg.GET("/observability/template/dashboard/:name", h.GetDashboardTemplate)
	rg.POST("/observability/template/dashboard", h.CheckIsSysADMIN, h.AddDashboardTemplates)
	rg.PUT("/observability/template/dashboard/:name", h.CheckIsSysADMIN, h.UpdateDashboardTemplates)
	rg.DELETE("/observability/template/dashboard/:name", h.CheckIsSysADMIN, h.DeleteDashboardTemplate)

	// exporter
	rg.GET("/observability/monitor/exporters/:name/schema", h.ExporterSchema)

	// otel
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/otel/appmonitor/services", h.CheckByClusterNamespace, h.OtelServices)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/otel/appmonitor/overview", h.CheckByClusterNamespace, h.OtelOverview)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/otel/appmonitor/services/:service_name/requests", h.CheckByClusterNamespace, h.OtelServiceRequests)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/otel/appmonitor/services/:service_name/operations", h.CheckByClusterNamespace, h.OtelServiceOperations)
	rg.GET("/observability/cluster/:cluster/namespaces/:namespace/otel/appmonitor/services/:service_name/traces", h.CheckByClusterNamespace, h.OtelServiceTraces)
}
