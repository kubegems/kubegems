package clusterhandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/server/define"
)

type ClusterHandler struct {
	define.ServerInterface
}

func (h *ClusterHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/cluster", h.CheckIsSysADMIN, h.ListCluster)
	rg.GET("/cluster/:cluster_id", h.CheckIsSysADMIN, h.RetrieveCluster)
	rg.POST("/cluster", h.CheckIsSysADMIN, h.PostCluster)
	rg.PUT("/cluster/:cluster_id", h.CheckIsSysADMIN, h.PutCluster)
	rg.DELETE("/cluster/:cluster_id", h.CheckIsSysADMIN, h.DeleteCluster)
	rg.GET("/cluster/_/status", h.CheckIsSysADMIN, h.ListClusterStatus)

	rg.GET("/cluster/:cluster_id/environment", h.CheckIsSysADMIN, h.ListClusterEnvironment)
	rg.GET("/cluster/:cluster_id/logqueryhistory", h.ListClusterLogQueryHistory)
	rg.GET("/cluster/:cluster_id/logqueryhistoryv2", h.ListClusterLogQueryHistoryv2)
	rg.GET("/cluster/:cluster_id/logquerysnapshot", h.ListClusterLogQuerySnapshot)
	rg.GET("/cluster/:cluster_id/quota", h.ListClusterQuota)

	rg.GET("/cluster/:cluster_id/plugins", h.CheckIsSysADMIN, h.ListPligins)
	rg.POST("/cluster/:cluster_id/plugins/:name/actions/enable", h.CheckIsSysADMIN, h.EnablePlugin)
	rg.POST("/cluster/:cluster_id/plugins/:name/actions/disable", h.CheckIsSysADMIN, h.DisablePlugin)
}
