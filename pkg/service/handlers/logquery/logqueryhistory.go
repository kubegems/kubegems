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

package logqueryhandler

import (
	"context"
	"strconv"

	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"

	"github.com/gin-gonic/gin"
)

// ListLogQueryHistory 列表 LogQueryHistory
//	@Tags			LogQueryHistory
//	@Summary		LogQueryHistory列表
//	@Description	LogQueryHistory列表
//	@Accept			json
//	@Produce		json
//	@Param			LogQL	query		string																			false	"LogQL"
//	@Param			preload	query		string																			false	"choices Cluster,Creator"
//	@Param			page	query		int																				false	"page"
//	@Param			size	query		int																				false	"page"
//	@Param			search	query		string																			false	"search in (LogQL)"
//	@Success		200		{object}	handlers.ResponseStruct{Data=handlers.PageData{List=[]models.LogQueryHistory}}	"LogQueryHistory"
//	@Router			/v1/logqueryhistory [get]
//	@Security		JWT
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
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// DeleteLogQueryHistory 删除 LogQueryHistory
//	@Tags			LogQueryHistory
//	@Summary		删除 LogQueryHistory
//	@Description	删除 LogQueryHistory
//	@Accept			json
//	@Produce		json
//	@Param			logqueryhistory_id	path		uint					true	"logqueryhistory_id"
//	@Success		204					{object}	handlers.ResponseStruct	"resp"
//	@Router			/v1/logqueryhistory/{logqueryhistory_id} [delete]
//	@Security		JWT
func (h *LogQueryHistoryHandler) DeleteLogQueryHistory(c *gin.Context) {
	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "logquery history")
	h.SetAuditData(c, action, module, "")
	lid, err := strconv.ParseUint(c.Param("logqueryhistory_id"), 10, 64)
	obj := models.LogQueryHistory{ID: uint(lid)}
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().WithContext(c.Request.Context()).Delete(&obj, uint(lid)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.NoContent(c, nil)
}

// DeleteLogQueryHistory 批量删除 LogQueryHistory
//	@Tags			LogQueryHistory
//	@Summary		批量删除 LogQueryHistory
//	@Description	批量删除 LogQueryHistory
//	@Accept			json
//	@Produce		json
//	@Param			logqueryhistory_id	query		uint					true	"lid"
//	@Success		204					{object}	handlers.ResponseStruct	"resp"
//	@Router			/v1/logqueryhistory [delete]
//	@Security		JWT
func (h *LogQueryHistoryHandler) BatchDeleteLogQueryHistory(c *gin.Context) {
	action := i18n.Sprintf(context.TODO(), "batch delete")
	module := i18n.Sprintf(context.TODO(), "logquery history")
	h.SetAuditData(c, action, module, "")
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
	if err := h.GetDB().WithContext(c.Request.Context()).Delete(&models.LogQueryHistory{}, "id in (?)", idints).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.NoContent(c, nil)
}

// PostLogQueryHistory 创建LogQueryHistory
//	@Tags			LogQueryHistory
//	@Summary		创建LogQueryHistory
//	@Description	创建LogQueryHistory
//	@Accept			json
//	@Produce		json
//	@Param			param	body		models.LogQueryHistory									true	"表单"
//	@Success		200		{object}	handlers.ResponseStruct{Data=models.LogQueryHistory}	"LogQueryHistory"
//	@Router			/v1/logqueryhistory [post]
//	@Security		JWT
func (h *LogQueryHistoryHandler) PostLogQueryHistory(c *gin.Context) {
	var obj models.LogQueryHistory
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	action := i18n.Sprintf(context.TODO(), "create")
	module := i18n.Sprintf(context.TODO(), "logquery history")
	h.SetAuditData(c, action, module, "")

	if err := h.GetDB().WithContext(c.Request.Context()).Create(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.Created(c, obj)
}
