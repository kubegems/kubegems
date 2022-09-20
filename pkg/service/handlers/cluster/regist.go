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

package clusterhandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
)

type ClusterHandler struct {
	base.BaseHandler
}

func (h *ClusterHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/cluster", h.CheckIsSysADMIN, h.ListCluster)
	rg.GET("/cluster/:cluster_id", h.CheckIsSysADMIN, h.RetrieveCluster)
	rg.POST("/cluster", h.CheckIsSysADMIN, h.PostCluster)
	rg.PUT("/cluster/:cluster_id", h.CheckIsSysADMIN, h.PutCluster)
	rg.DELETE("/cluster/:cluster_id", h.CheckIsSysADMIN, h.DeleteCluster)
	rg.GET("/cluster/_/status", h.CheckIsSysADMIN, h.ListClusterStatus)

	rg.POST("/cluster/validate-kubeconfig", h.CheckIsSysADMIN, h.ValidateKubeConfig)

	rg.GET("/cluster/:cluster_id/environment", h.CheckIsSysADMIN, h.ListClusterEnvironment)
	rg.GET("/cluster/:cluster_id/logqueryhistory", h.ListClusterLogQueryHistory)
	rg.GET("/cluster/:cluster_id/logqueryhistoryv2", h.ListClusterLogQueryHistoryv2)
	rg.GET("/cluster/:cluster_id/logquerysnapshot", h.ListClusterLogQuerySnapshot)
	rg.GET("/cluster/:cluster_id/quota", h.ListClusterQuota)
}
