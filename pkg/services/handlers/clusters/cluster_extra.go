package clusterhandler

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/agent/apis/types"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/utils/agents"
)

func (h *Handler) ListEnvironment(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ol := forms.EnvironmentDetailList{}
	cluster := forms.ClusterCommon{
		Name: req.PathParameter("cluster"),
	}
	if err := h.Model().List(
		ctx,
		ol.Object(),
		client.BelongTo(cluster.Object()),
		client.Preloads([]string{"Project", "Cluster", "Creator", "Applications", "Users"}),
	); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(ol.Object(), ol.DataPtr()))
}

func (h *Handler) ListLogQueryHistory(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ol := forms.LogQueryHistoryCommonList{}
	cluster := forms.ClusterCommon{
		Name: req.PathParameter("cluster"),
	}
	if err := h.Model().List(
		ctx,
		ol.Object(),
		client.BelongTo(cluster.Object()),
	); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(ol.Object(), ol.DataPtr()))
}

func (h *Handler) ListLogQuerySnapshot(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ol := forms.LogQuerySnapshotCommonList{}
	cluster := forms.ClusterCommon{
		Name: req.PathParameter("cluster"),
	}
	if err := h.Model().List(
		ctx,
		ol.Object(),
		client.BelongTo(cluster.Object()),
	); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(ol.Object(), ol.DataPtr()))
}

type ClusterQuota struct {
	Version        string                          `json:"version"`
	OversoldConfig []byte                          `json:"oversoldConfig"`
	Resoruces      types.ClusterResourceStatistics `json:"resources"`
	Workloads      types.ClusterWorkloadStatistics `json:"workloads"`
}

func (h *Handler) GetClusterQuotaStastic(req *restful.Request, resp *restful.Response) {
	h.cluster(req, resp, func(ctx context.Context, cluster string, cli agents.Client) (interface{}, error) {
		clusterData, err := h.getClusterDetail(ctx, req.PathParameter("cluster"))
		if err != nil {
			return nil, err
		}
		resources := types.ClusterResourceStatistics{}
		if err := cli.Extend().ClusterResourceStatistics(ctx, &resources); err != nil {
			return nil, err
		}
		workloads := types.ClusterWorkloadStatistics{}
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
