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

package apis

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/utils/statistics"
)

type StatisticsHandler struct {
	C cluster.Interface
}

// @Tags        Agent.V1
// @Summary     获取集群内各种workload的统计
// @Description 获取集群内各种workload的统计
// @Accept      json
// @Produce     json
// @Param       cluster path     string                                                     true "cluster"
// @Success     200     {object} handlers.ResponseStruct{Data=object} "counter"
// @Router      /v1/proxy/cluster/{cluster}/custom/statistics.system/v1/workloads [get]
// @Security    JWT
func (sth *StatisticsHandler) ClusterWorkloadStatistics(c *gin.Context) {
	ret := statistics.GetWorkloadsStatistics(c.Request.Context(), sth.C.GetClient())
	OK(c, ret)
}

// ClusterResourceStatistics  获取集群级别资源统计
// @Tags        Agent.V1
// @Summary     获取集群级别资源统计
// @Description 获取集群级别资源统计
// @Accept      json
// @Produce     json
// @Param       cluster path     string                               true "cluster"
// @Success     200     {object} handlers.ResponseStruct{Data=object} "counter"
// @Router      /v1/proxy/cluster/{cluster}/custom/statistics.system/v1/resources [get]
// @Security    JWT
func (sth *StatisticsHandler) ClusterResourceStatistics(c *gin.Context) {
	clusterResourceStatistics := statistics.GetClusterResourceStatistics(c, sth.C.GetClient())
	OK(c, clusterResourceStatistics)
}

// ClusterResourceStatistics  获取集群级别统计
// @Tags        Agent.V1
// @Summary     获取集群级别统计
// @Description 获取集群级别统计
// @Accept      json
// @Produce     json
// @Param       cluster path     string                               true "cluster"
// @Success     200     {object} handlers.ResponseStruct{Data=statistics.ClusterStatistics} "counter"
// @Router      /v1/proxy/cluster/{cluster}/custom/statistics.system/v1/all [get]
// @Security    JWT
func (sth *StatisticsHandler) ClusterStatistics(c *gin.Context) {
	clusterResourceStatistics := statistics.GetClusterAllStatistics(c, sth.C.GetClient(), sth.C.Discovery())
	OK(c, clusterResourceStatistics)
}
