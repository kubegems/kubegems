package clusterhandler

import (
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/services/handlers"
)

type ClusterInfoResp struct {
	handlers.PageBase
	Data models.ClusterSimple `json:"data"`
}

type ClusterListResp struct {
	handlers.PageBase
	Data []models.ClusterSimple `json:"list"`
}

type LogQueryHistoryListResp struct {
	handlers.PageBase
	Data []models.LogQueryHistory `json:"list"`
}

type LogQuerySnapshotListResp struct {
	handlers.PageBase
	Data []models.LogQuerySnapshot `json:"list"`
}
