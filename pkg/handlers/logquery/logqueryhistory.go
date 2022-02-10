package logqueryhandler

import (
	"strconv"

	"github.com/kubegems/gems/pkg/handlers"
	"github.com/kubegems/gems/pkg/models"

	"github.com/gin-gonic/gin"
)

// ListLogQueryHistory 列表 LogQueryHistory
// @Tags LogQueryHistory
// @Summary LogQueryHistory列表
// @Description LogQueryHistory列表
// @Accept json
// @Produce json
// @Param LogQL query string false "LogQL"
// @Param preload query string false "choices Cluster,Creator"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (LogQL)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.LogQueryHistory}} "LogQueryHistory"
// @Router /v1/logqueryhistory [get]
// @Security JWT
func (h *LogQueryHistoryHandler) ListLogQueryHistory(c *gin.Context) {
	var list []models.LogQueryHistory
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "LogQueryHistory",
		SearchFields:  []string{"LogQL"},
		PreloadFields: []string{"Cluster", "Creator"},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// DeleteLogQueryHistory 删除 LogQueryHistory
// @Tags LogQueryHistory
// @Summary 删除 LogQueryHistory
// @Description 删除 LogQueryHistory
// @Accept json
// @Produce json
// @Param logqueryhistory_id path uint true "logqueryhistory_id"
// @Success 204 {object} handlers.ResponseStruct "resp"
// @Router /v1/logqueryhistory/{logqueryhistory_id} [delete]
// @Security JWT
func (h *LogQueryHistoryHandler) DeleteLogQueryHistory(c *gin.Context) {
	h.SetAuditData(c, "删除", "日志查询历史", "")
	lid, err := strconv.ParseUint(c.Param("logqueryhistory_id"), 10, 64)
	obj := models.LogQueryHistory{ID: uint(lid)}
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Delete(&obj, uint(lid)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.NoContent(c, nil)
}

// DeleteLogQueryHistory 批量删除 LogQueryHistory
// @Tags LogQueryHistory
// @Summary 批量删除 LogQueryHistory
// @Description 批量删除 LogQueryHistory
// @Accept json
// @Produce json
// @Param logqueryhistory_id query uint true "lid"
// @Success 204 {object} handlers.ResponseStruct "resp"
// @Router /v1/logqueryhistory [delete]
// @Security JWT
func (h *LogQueryHistoryHandler) BatchDeleteLogQueryHistory(c *gin.Context) {
	h.SetAuditData(c, "批量删除", "日志查询历史", "")
	ids := c.QueryArray("lid")
	if len(ids) == 0 {
		handlers.NoContent(c, nil)
		return
	}
	var idints []uint
	for _, idstr := range ids {
		i, e := strconv.ParseUint(idstr, 10, 64)
		if e != nil {
			continue
		}
		idints = append(idints, uint(i))
	}
	if len(idints) == 0 {
		handlers.NoContent(c, nil)
		return
	}
	if err := h.GetDB().Delete(&models.LogQueryHistory{}, "id in (?)", idints).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.NoContent(c, nil)
}

// PostLogQueryHistory 创建LogQueryHistory
// @Tags LogQueryHistory
// @Summary 创建LogQueryHistory
// @Description 创建LogQueryHistory
// @Accept json
// @Produce json
// @Param param body models.LogQueryHistory true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.LogQueryHistory} "LogQueryHistory"
// @Router /v1/logqueryhistory [post]
// @Security JWT
func (h *LogQueryHistoryHandler) PostLogQueryHistory(c *gin.Context) {
	var obj models.LogQueryHistory
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "创建", "日志查询历史", "")

	if err := h.GetDB().Create(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.Created(c, obj)
}
