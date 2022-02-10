package tenanthandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/server/define"
)

// TenantHandler 租户相关 Handler
type TenantHandler struct {
	define.ServerInterface
}

func (h *TenantHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/tenant", h.CheckIsSysADMIN, h.ListTenant)
	rg.GET("/tenant/:tenant_id", h.CheckByTenantID, h.RetrieveTenant)
	rg.POST("/tenant", h.CheckIsSysADMIN, h.PostTenant)
	rg.PUT("/tenant/:tenant_id", h.CheckByTenantID, h.PutTenant)
	rg.DELETE("/tenant/:tenant_id", h.CheckByTenantID, h.DeleteTenant)

	rg.GET("/tenant/:tenant_id/user", h.CheckByTenantID, h.ListTenantUser)
	rg.GET("/tenant/:tenant_id/user/:user_id", h.CheckByTenantID, h.RetrieveTenantUser)
	rg.POST("/tenant/:tenant_id/user", h.CheckByTenantID, h.PostTenantUser)
	rg.PUT("/tenant/:tenant_id/user/:user_id", h.CheckByTenantID, h.PutTenantUser)
	rg.DELETE("/tenant/:tenant_id/user/:user_id", h.CheckByTenantID, h.DeleteTenantUser)

	rg.GET("/tenant/:tenant_id/project", h.CheckByTenantID, h.ListTenantProject)
	rg.GET("/tenant/:tenant_id/project/:project_id", h.CheckByTenantID, h.RetrieveTenantProject)
	rg.POST("/tenant/:tenant_id/project", h.CheckByTenantID, h.PostTenantProject)

	rg.GET("/tenant/:tenant_id/tenantresourcequota", h.CheckByTenantID, h.ListTenantTenantResourceQuota)
	rg.POST("/tenant/:tenant_id/tenantresourcequota", h.CheckByTenantID, h.PostTenantTenantResourceQuota)
	rg.GET("/tenant/:tenant_id/tenantresourcequota/:tenantresourcequota_id", h.CheckByTenantID, h.RetrieveTenantTenantResourceQuota)

	rg.PUT("/tenant/:tenant_id/action/enable", h.CheckByTenantID, h.EnableTenant)
	rg.PUT("/tenant/:tenant_id/action/disable", h.CheckByTenantID, h.DisableTenant)
	rg.POST("/tenant/:tenant_id/action/networkisolate", h.CheckByTenantID, h.TenantSwitch)

	rg.GET("/tenant/:tenant_id/environment_with_quotas", h.CheckByTenantID, h.TenantEnvironments)
	rg.GET("/tenant/:tenant_id/environment", h.CheckByTenantID, h.ListEnvironment)
	rg.GET("/tenant/:tenant_id/statistics", h.CheckByTenantID, h.TenantStatistics)

	rg.PUT("/tenant/:tenant_id/tenantresourcequota/:cluster_id", h.CheckByTenantID, h.PutTenantTenantResourceQuota)
	rg.DELETE("/tenant/:tenant_id/tenantresourcequota/:cluster_id", h.CheckByTenantID, h.DeleteTenantResourceQuota)
	rg.POST("/tenant/:tenant_id/cluster/:cluster_id/resourceApply", h.CheckByTenantID, h.CreateTenantResourceQuotaApply)
	rg.GET("/tenant/:tenant_id/tenantresourcequotaapply/:tenantresourcequotaapply_id", h.CheckByTenantID, h.GetTenantTenantResourceQuotaApply)

	rg.GET("/tenant/:tenant_id/cluster/:cluster_id/tenantgateways", h.CheckByTenantID, h.ListTenantGateway)
	rg.GET("/tenant/:tenant_id/cluster/:cluster_id/tenantgateways/:name", h.CheckByTenantID, h.GetTenantGateway)
	rg.POST("/tenant/:tenant_id/cluster/:cluster_id/tenantgateways", h.CheckByTenantID, h.CreateTenantGateway)
	rg.PUT("/tenant/:tenant_id/cluster/:cluster_id/tenantgateways/:name", h.CheckByTenantID, h.UpdateTenantGateway)
	rg.DELETE("/tenant/:tenant_id/cluster/:cluster_id/tenantgateways/:name", h.CheckByTenantID, h.DeleteTenantGateway)
	rg.GET("/tenant/:tenant_id/cluster/:cluster_id/tenantgateways/:name/addresses", h.CheckByTenantID, h.GetObjectTenantGatewayAddr)
}
