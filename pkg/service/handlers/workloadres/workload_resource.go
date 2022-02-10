package workloadreshandler

import (
	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/service/handlers"

	"github.com/gin-gonic/gin"
)

var (
	FilterFields   = []string{"ClusterName", "Namespace", "Type", "CPULimitStdvar", "MemoryLimitStdvar"}
	PreloadFields  = []string{"Containers"}
	OrderFields    = []string{}
	ModelName      = "Workload"
	PrimaryKeyName = "workload_id"
)

// ListWorkload 列表 Workload
// @Tags ResourceList
// @Summary Workload列表
// @Description Workload列表
// @Accept json
// @Produce json
// @Param cluster query string false "cluster"
// @Param namespace query string false "namespace"
// @Param workloadtype query string false "workloadtype"
// @Success 200 {object} handlers.ResponseStruct{Data=[]models.Workload} "Workload"
// @Router /v1/resources/workload [get]
// @Security JWT
func (h *WorkloadHandler) ListWorkload(c *gin.Context) {
	cluster := c.Query("cluster")
	namespace := c.Query("namespace")
	workloadtype := c.Query("workloadtype")
	var workloads []*models.Workload
	tx := h.GetDB().
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
// @Tags ResourceList
// @Summary Workload删除
// @Description Workload删除
// @Accept json
// @Produce json
// @Param workload_id path uint true "workload_id"
// @Success 204 {object} handlers.ResponseStruct resp
// @Router /v1/resources/workload/{workload_id} [delete]
// @Security JWT
func (h *WorkloadHandler) DeleteWorkload(c *gin.Context) {
	var obj models.Workload
	if err := h.GetDB().Delete(&obj, c.Param("workload_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.NoContent(c, nil)
}
