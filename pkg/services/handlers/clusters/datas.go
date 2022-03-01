package clusterhandler

import (
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/services/handlers"
)

type ClusterInfoResp struct {
	handlers.RespBase
	Data models.ClusterSimple `json:"data"`
}

type ClusterListResp struct {
	handlers.ListBase
	Data []models.ClusterSimple `json:"list"`
}

type LogQueryHistoryListResp struct {
	handlers.ListBase
	Data []models.LogQueryHistory `json:"list"`
}

type LogQuerySnapshotListResp struct {
	handlers.ListBase
	Data []models.LogQuerySnapshot `json:"list"`
}

type ClusterResp struct {
	handlers.RespBase
	Data models.Cluster `json:"data"`
}

type EnvironmentListResp struct {
	handlers.ListBase
	Data []models.Environment `json:"list"`
}

type ClusterQuotaResp struct {
	handlers.RespBase
	Data ClusterQuota `json:"data"`
}

type ClusterStatusMapResp struct {
	handlers.RespBase
	Data ClusterStatusMap `json:"data"`
}
