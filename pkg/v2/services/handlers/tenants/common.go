package tenanthandler

import (
	"context"

	"kubegems.io/kubegems/pkg/v2/models"
)

func (h *Handler) getTenant(ctx context.Context, name string) (*models.TenantCommon, error) {
	tenant := &models.TenantCommon{}
	err := h.DB().WithContext(ctx).First(tenant, "name = ?", name).Error
	return tenant, err
}

func (h *Handler) getUser(ctx context.Context, name string) (*models.UserCommon, error) {
	user := &models.UserCommon{}
	err := h.DB().WithContext(ctx).First(user, "username = ?", name).Error
	return user, err
}

func (h *Handler) getProject(ctx context.Context, tenant, name string) (*models.Project, error) {
	project := &models.Project{}
	err := h.DB().WithContext(ctx).Joins("LEFT JOIN tenants on tenants.id = projects.tenant_id").Where("projects.name = ?", name).First(project).Error
	return project, err
}

func (h *Handler) getCluster(ctx context.Context, name string) (*models.Cluster, error) {
	cluster := &models.Cluster{}
	err := h.DB().WithContext(ctx).First(cluster, "name = ?", name).Error
	return cluster, err
}

func (h *Handler) getTenantResourceQuota(ctx context.Context, tenant, cluster string) (*models.TenantResourceQuota, error) {
	trq := &models.TenantResourceQuota{}
	err := h.DB().WithContext(ctx).
		Joins("LEFT JOIN tenants on tenants.id = tenant_resource_quotas.tenant_id").
		Joins("LEFT JOIN clusters on clusters.id = tenant_resource_quotas.cluster_id").
		Where("tenants.name = ? and clusters.name", tenant, cluster).
		First(trq).Error
	return trq, err
}

func (h *Handler) getProjectEnvironment(ctx context.Context, project, environment string) (*models.Environment, error) {
	env := &models.Environment{}
	err := h.DB().WithContext(ctx).
		Joins("LEFT JOIN projects on projects.id = environments.project_id").
		Where("projects.name = ?", project).
		First(env).Error
	return env, err

}
