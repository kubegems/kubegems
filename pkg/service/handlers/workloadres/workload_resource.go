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

package workloadreshandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
)

var (
	FilterFields   = []string{"ClusterName", "Namespace", "Type", "CPULimitStdvar", "MemoryLimitStdvar"}
	PreloadFields  = []string{"Containers"}
	OrderFields    = []string{}
	ModelName      = "Workload"
	PrimaryKeyName = "workload_id"
)

// ListWorkload 列表 Workload
//
//	@Tags			ResourceList
//	@Summary		Workload列表
//	@Description	Workload列表
//	@Accept			json
//	@Produce		json
//	@Param			cluster			query		string													false	"cluster"
//	@Param			namespace		query		string													false	"namespace"
//	@Param			workloadtype	query		string													false	"workloadtype"
//	@Success		200				{object}	response.Page[models.Workload]{Data=[]models.Workload}	"Workload"
//	@Router			/v1/resources/workload [get]
//	@Security		JWT
func (h *WorkloadHandler) ListWorkload(c *gin.Context) {
	cluster := c.Query("cluster")
	namespace := c.Query("namespace")
	workloadtype := c.Query("workloadtype")
	var workloads []*models.Workload
	tx := h.GetDB().WithContext(c.Request.Context()).
		Preload("Containers").
		Where("cpu_limit_stdvar = 0 and memory_limit_stdvar = 0 and cluster_name = ? and type = ?", cluster, workloadtype)
	if namespace != "_all" {
		tx.Where("namespace = ?", namespace)
	}

	if err := tx.Find(&workloads).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	ret := []*models.Workload{}
	for _, v := range workloads {
		v.AddNotice()
		if len(v.Notice.Conditions) > 0 {
			ret = append(ret, v)
		}
	}
	handlers.OK(c, ret)
}

// Delete 删除 Workload
//
//	@Tags			ResourceList
//	@Summary		Workload删除
//	@Description	Workload删除
//	@Accept			json
//	@Produce		json
//	@Param			workload_id	path		uint											true	"workload_id"
//	@Success		204			{object}	handlers.ResponseStruct{Data=models.Workload}	"workload"
//	@Router			/v1/resources/workload/{workload_id} [delete]
//	@Security		JWT
func (h *WorkloadHandler) DeleteWorkload(c *gin.Context) {
	var obj models.Workload
	if err := h.GetDB().WithContext(c.Request.Context()).Delete(&obj, c.Param("workload_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.NoContent(c, nil)
}
