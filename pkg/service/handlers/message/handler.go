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

package messagehandler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/msgbus"
)

type MessageRet []models.Message

func (a MessageRet) Len() int           { return len(a) }
func (a MessageRet) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a MessageRet) Less(i, j int) bool { return a[i].CreatedAt.After(a[j].CreatedAt) }

// ListMessage 获取我的消息列表
// @Tags        Message
// @Summary     获取我的消息列表
// @Description 获取我的消息列表
// @Accept      json
// @Produce     json
// @Param       page         query    int                                                      false "page"
// @Param       size         query    int                                                      false "page"
// @Param       is_read      query    bool                                                     false "是否已读，不指定则是所有"
// @Param       message_type query    string                                                   false "消息类型(message、alert、approve)，不指定则是所有"
// @Success     200          {object} handlers.ResponseStruct{Data=[]models.UserMessageStatus} "messages"
// @Router      /v1/message [get]
// @Security    JWT
func (h *MessageHandler) ListMessage(c *gin.Context) {
	user, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	messageType := c.Query("message_type")
	q := h.GetDB().WithContext(c.Request.Context()).Where("user_id = ?", user.GetID())

	switch messageType {
	case string(msgbus.Message), string(msgbus.Approve):
		q.Preload("Message").
			Joins("join messages on user_message_statuses.message_id = messages.id and messages.message_type = ?", messageType).
			Where("message_id is not null").
			Order("user_message_statuses.id desc")

	case string(msgbus.Alert):
		q.Preload("AlertMessage.AlertInfo").
			Where("alert_message_id is not null").
			Order("id desc")
	default:
		q.Preload("Message").Preload("AlertMessage.AlertInfo").
			Where("message_id is not null or alert_message_id is not null").
			Order("id desc")
	}

	isReadStr := c.Query("is_read")
	if isReadStr != "" {
		isRead, _ := strconv.ParseBool(isReadStr)
		q.Where("is_read = ?", isRead)
	}

	var total int64
	var allMsgStatuses []models.UserMessageStatus
	// 总数
	if err := q.Model(&models.UserMessageStatus{}).Count(&total).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	// 查所有id
	if err := q.Limit(size).Offset((page - 1) * size).Find(&allMsgStatuses).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	// 合并、排序、返回
	ret := make(MessageRet, len(allMsgStatuses))
	for i := range allMsgStatuses {
		if allMsgStatuses[i].Message != nil {
			ret[i] = *allMsgStatuses[i].Message
		} else if allMsgStatuses[i].AlertMessage != nil {
			ret[i] = allMsgStatuses[i].AlertMessage.ToNormalMessage()
		}
		ret[i].IsRead = allMsgStatuses[i].IsRead
	}
	sort.Sort(ret)
	handlers.OK(c, handlers.Page(total, ret, int64(page), int64(size)))
}

// ReadMessage 获取消息详情
// @Tags        Message
// @Summary     获取消息详情
// @Description 获取消息详情,获取之后将自动标记成了已读
// @Accept      json
// @Produce     json
// @Param       message_id   path     uint                                         true "message_id"
// @Param       message_type path     uint                                         true "消息类型(message/alert)"
// @Success     200          {object} handlers.ResponseStruct{Data=models.Message} "messages"
// @Router      /v1/message/{message_id} [put]
// @Security    JWT
func (h *MessageHandler) ReadMessage(c *gin.Context) {
	msgID := c.Param("message_id")
	msgType := c.Query("message_type")
	if msgID != "_all" &&
		!(msgType == string(msgbus.Message) ||
			msgType == string(msgbus.Alert) ||
			msgType == string(msgbus.Approve)) {
		msg := i18n.Sprintf(c, "message type %s is invalid", msgType)
		handlers.NotOK(c, fmt.Errorf(msg))
		return
	}
	user, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, nil)
		return
	}

	ctx := c.Request.Context()
	readQuery := h.GetDB().WithContext(ctx).Model(models.UserMessageStatus{}).Where("user_id = ?", user.GetID())
	details := models.Message{}
	if msgID != "_all" {
		switch msgType {
		case string(msgbus.Message), string(msgbus.Approve):
			readQuery.Where("message_id = ?", msgID)
			// 返回
			if err := h.GetDB().WithContext(ctx).First(&details, "id = ?", msgID).Error; err != nil {
				handlers.NotOK(c, err)
				return
			}
		case string(msgbus.Alert):
			readQuery.Where("alert_message_id = ?", msgID)
			// 返回
			alertmsg := models.AlertMessage{}
			if err := h.GetDB().WithContext(ctx).First(&alertmsg, "id = ?", msgID).Error; err != nil {
				handlers.NotOK(c, err)
				return
			}
			if err := h.GetDB().WithContext(ctx).First(alertmsg.AlertInfo, "fingerprint = ?", alertmsg.Fingerprint).Error; err != nil {
				handlers.NotOK(c, err)
				return
			}
			labels := map[string]string{}
			json.Unmarshal(alertmsg.AlertInfo.Labels, &labels)

			_, isMonitor := labels["prometheus"]
			pos, err := h.GetDataBase().GetAlertPosition(alertmsg.AlertInfo.ClusterName, alertmsg.AlertInfo.Namespace, alertmsg.AlertInfo.Name, isMonitor)
			if err != nil {
				handlers.NotOK(c, err)
				return
			}
			details.ID = alertmsg.ID
			details.Content, _ = json.Marshal(pos)
			details.CreatedAt = *alertmsg.CreatedAt
			details.MessageType = string(msgbus.Alert)
			details.Title = alertmsg.Message
		}
	}
	if err := readQuery.Updates(map[string]interface{}{"is_read": true}).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, details)
}
