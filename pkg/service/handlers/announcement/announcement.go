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

package announcement

import (
	"strconv"
	"time"

	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	"kubegems.io/kubegems/pkg/service/models"

	"github.com/gin-gonic/gin"
)

var (
	SearchFields   = []string{"Username", "Email"}
	FilterFields   = []string{"Username"}
	PreloadFields  = []string{"Tenants", "SystemRole"}
	OrderFields    = []string{"Username", "ID"}
	ModelName      = "User"
	PrimaryKeyName = "user_id"
)

// ListAnnouncement 公告列表
// @Tags        Announcement
// @Summary     公告列表
// @Description 公告列表
// @Accept      json
// @Produce     json
// @Param       active query    bool                                                                        true "是否为活跃中的公告"
// @Success     200    {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Announcement}} "resp"
// @Router      /v1/announcement [get]
// @Security    JWT
func (h *AnnouncementHandler) ListAnnouncement(c *gin.Context) {
	active, _ := strconv.ParseBool(c.Query("active"))
	list := []models.Announcement{}
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model: "Announcement",
	}
	if active {
		now := time.Now()
		cond.Where = append(cond.Where,
			handlers.Args("start_at <= ? and end_at >= ?", now, now),
		)
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// GetAnnouncement 获取单个公告
// @Tags        Announcement
// @Summary     获取单个公告
// @Description 获取单个公告
// @Accept      json
// @Produce     json
// @Param       id  path     uint                                              true "公告 id"
// @Success     200 {object} handlers.ResponseStruct{Data=models.Announcement} "resp"
// @Router      /v1/announcement/{id} [get]
// @Security    JWT
func (h *AnnouncementHandler) GetAnnouncement(c *gin.Context) {
	ret := models.Announcement{}
	if err := h.GetDB().First(&ret, c.Param("id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, ret)
}

// PostAnnouncement 发布公告
// @Tags        Announcement
// @Summary     发布公告
// @Description 发布公告
// @Accept      json
// @Produce     json
// @Param       form body     models.Announcement                  true "公告内容"
// @Success     200  {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/announcement [post]
// @Security    JWT
func (h *AnnouncementHandler) PostAnnouncement(c *gin.Context) {
	req := models.Announcement{}
	if err := c.Bind(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.setAnnounceDuration(&req)
	if req.EndAt.Before(*req.StartAt) {
		handlers.NotOK(c, i18n.Errorf(c, "site announcement period is invalid, the end time is earlier than the start time"))
		return
	}
	if err := h.GetDB().Create(&req).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// PutAnnouncement 更新公告
// @Tags        Announcement
// @Summary     更新公告
// @Description 更新公告
// @Accept      json
// @Produce     json
// @Param       id   path     uint                                 true "公告 id"
// @Param       form body     models.Announcement                  true "公告内容"
// @Success     200  {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/announcement/{id} [put]
// @Security    JWT
func (h *AnnouncementHandler) PutAnnouncement(c *gin.Context) {
	req := models.Announcement{}
	if err := c.Bind(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.setAnnounceDuration(&req)
	if req.EndAt.Before(*req.StartAt) {
		handlers.NotOK(c, i18n.Errorf(c, "site announcement period is invalid, the end time is earlier than the start time"))
		return
	}
	if err := h.GetDB().Updates(&req).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteAnnouncement 删除公告
// @Tags        Announcement
// @Summary     删除公告
// @Description 删除公告
// @Accept      json
// @Produce     json
// @Param       id  path     uint                                 true "公告 id"
// @Success     200 {object} handlers.ResponseStruct{Data=string} "resp"
// @Router      /v1/announcement/{id} [delete]
// @Security    JWT
func (h *AnnouncementHandler) DeleteAnnouncement(c *gin.Context) {
	req := models.Announcement{}
	if err := h.GetDB().Delete(&req, c.Param("id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

func (h *AnnouncementHandler) setAnnounceDuration(announce *models.Announcement) {
	timedelta := time.Hour * 24
	if announce.StartAt == nil && announce.EndAt == nil {
		startTime := time.Now()
		endTime := startTime.Add(timedelta)
		announce.StartAt = &startTime
		announce.EndAt = &endTime
	}
	if announce.StartAt == nil && announce.EndAt != nil {
		startTime := announce.EndAt.Add(-timedelta)
		announce.StartAt = &startTime
	}
	if announce.StartAt != nil && announce.EndAt == nil {
		endTime := announce.StartAt.Add(timedelta)
		announce.EndAt = &endTime
	}
}

type AnnouncementHandler struct {
	base.BaseHandler
}

func (h *AnnouncementHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/announcement", h.ListAnnouncement)
	rg.GET("/announcement/:id", h.GetAnnouncement)
	rg.POST("/announcement", h.CheckIsSysADMIN, h.PostAnnouncement)
	rg.PUT("/announcement/:id", h.CheckIsSysADMIN, h.PutAnnouncement)
	rg.DELETE("/announcement/:id", h.CheckIsSysADMIN, h.DeleteAnnouncement)
}
