package clusterhandler

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/statistics"
	"kubegems.io/pkg/v2/models"
	"kubegems.io/pkg/v2/services/handlers"
)

func (h *Handler) ListEnvironment(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	envlist := &[]models.Environment{}
	if err := h.DB().WithContext(ctx).
		Joins("LEFT JOIN clusters on clusters.id = environments.cluster_id").
		Where("clusters.name = ?", req.PathParameter("cluster")).
		Find(envlist).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, envlist)
}

func (h *Handler) ListLogQueryHistory(req *restful.Request, resp *restful.Response) {
	ol := &[]models.LogQueryHistory{}
	scopes := []func(*gorm.DB) *gorm.DB{
		handlers.ScopeTable(ol),
		handlers.ScopeBelongViaField(&models.Cluster{}, ol, handlers.WhereNameEqual(req.PathParameter("cluster")), "cluster_id"),
		handlers.ScopeOrder(req, []string{"create_at"}),
	}
	var total int64
	if err := h.DBWithContext(req).Scopes(scopes...).Count(&total).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	scopes = append(scopes, handlers.ScopePageSize(req))
	db := h.DBWithContext(req).Scopes(scopes...).Find(ol)
	if err := db.Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(db, total, ol))
}

func (h *Handler) ListLogQuerySnapshot(req *restful.Request, resp *restful.Response) {
	ol := &[]models.LogQuerySnapshot{}
	scopes := []func(*gorm.DB) *gorm.DB{
		handlers.ScopeTable(ol),
		handlers.ScopeBelongViaField(&models.Cluster{}, ol, handlers.WhereNameEqual(req.PathParameter("cluster")), "cluster_id"),
		handlers.ScopeOrder(req, []string{"create_at"}),
	}
	var total int64
	if err := h.DBWithContext(req).Scopes(scopes...).Count(&total).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	scopes = append(scopes, handlers.ScopePageSize(req))
	db := h.DBWithContext(req).Scopes(scopes...).Find(ol)
	if err := db.Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(db, total, ol))
}

type ClusterQuota struct {
	Version        string                          `json:"version"`
	OversoldConfig []byte                          `json:"oversoldConfig"`
	Resoruces      statistics.ClusterResourceStatistics `json:"resources"`
	Workloads      statistics.ClusterWorkloadStatistics `json:"workloads"`
}

func (h *Handler) GetClusterQuotaStastic(req *restful.Request, resp *restful.Response) {
	h.cluster(req, resp, func(ctx context.Context, cluster string, cli agents.Client) (interface{}, error) {
		clusterData, err := h.getCluster(ctx, req.PathParameter("cluster"))
		if err != nil {
			return nil, err
		}
		resources := statistics.ClusterResourceStatistics{}
		if err := cli.Extend().ClusterResourceStatistics(ctx, &resources); err != nil {
			return nil, err
		}
		workloads := statistics.ClusterWorkloadStatistics{}
		if err := cli.Extend().ClusterWorkloadStatistics(ctx, &workloads); err != nil {
			return nil, err
		}
		return ClusterQuota{
			Version:        clusterData.Version,
			Resoruces:      resources,
			OversoldConfig: clusterData.OversoldConfig,
			Workloads:      workloads,
		}, nil
	})
}
