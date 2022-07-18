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
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"

	"github.com/gin-gonic/gin"
)

// ListLogQuerySnapshot 列表 LogQuerySnapshot
// @Tags         LogQuerySnapshot
// @Summary      LogQuerySnapshot列表
// @Description  LogQuerySnapshot列表
// @Accept       json
// @Produce      json
// @Param        SnapshotName  query     string                                                                           false  "SnapshotName"
// @Param        preload       query     string                                                                           false  "choices Cluster,Creator"
// @Param        page          query     int                                                                              false  "page"
// @Param        size          query     int                                                                              false  "page"
// @Param        search        query     string                                                                           false  "search in (SnapshotName)"
// @Success      200           {object}  handlers.ResponseStruct{Data=handlers.PageData{List=[]models.LogQuerySnapshot}}  "LogQuerySnapshot"
// @Router       /v1/logquerysnapshot [get]
// @Security     JWT
func (h *LogQuerySnapshotHandler) ListLogQuerySnapshot(c *gin.Context) {
	var list []models.LogQuerySnapshot
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "LogQuerySnapshot",
		SearchFields:  []string{"SnapshotName"},
		PreloadFields: []string{"Cluster", "Creator"},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveLogQuerySnapshot LogQuerySnapshot详情
// @Tags         LogQuerySnapshot
// @Summary      LogQuerySnapshot详情
// @Description  get LogQuerySnapshot详情
// @Accept       json
// @Produce      json
// @Param        logquerysnapshot_id  path      uint                                                   true  "logquerysnapshot_id"
// @Success      200                  {object}  handlers.ResponseStruct{Data=models.LogQuerySnapshot}  "LogQuerySnapshot"
// @Router       /v1/logquerysnapshot/{logquerysnapshot_id} [get]
// @Security     JWT
func (h *LogQuerySnapshotHandler) RetrieveLogQuerySnapshot(c *gin.Context) {
	var obj models.LogQuerySnapshot
	if err := h.GetDB().First(&obj, c.Param("logquerysnapshot_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// DeleteLogQuerySnapshot 删除 LogQuerySnapshot
// @Tags         LogQuerySnapshot
// @Summary      删除 LogQuerySnapshot
// @Description  删除 LogQuerySnapshot
// @Accept       json
// @Produce      json
// @Param        logquerysnapshot_id  path      uint                     true  "logquerysnapshot_id"
// @Success      204                  {object}  handlers.ResponseStruct  "resp"
// @Router       /v1/logquerysnapshot/{logquerysnapshot_id} [delete]
// @Security     JWT
func (h *LogQuerySnapshotHandler) DeleteLogQuerySnapshot(c *gin.Context) {
	var obj models.LogQuerySnapshot
	if err := h.GetDB().First(&obj, c.Param("logquerysnapshot_id")).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	h.SetAuditData(c, "删除", "日志快照", obj.SnapshotName)
	if err := h.GetDB().Delete(&obj, c.Param("logquerysnapshot_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.NoContent(c, nil)
}

// PostLogQuerySnapshot 创建LogQuerySnapshot
// @Tags         LogQuerySnapshot
// @Summary      创建LogQuerySnapshot
// @Description  创建LogQuerySnapshot
// @Accept       json
// @Produce      json
// @Param        param  body      models.LogQuerySnapshot                                true  "表单"
// @Success      200    {object}  handlers.ResponseStruct{Data=models.LogQuerySnapshot}  "LogQuerySnapshot"
// @Router       /v1/logquerysnapshot [post]
// @Security     JWT
func (h *LogQuerySnapshotHandler) PostLogQuerySnapshot(c *gin.Context) {
	var obj models.LogQuerySnapshot
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "创建", "日志快照", obj.SnapshotName)
	if err := h.GetDB().Create(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.Created(c, obj)
}
