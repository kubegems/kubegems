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

package observability

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

func (h *ObservabilityHandler) getChannelReq(c *gin.Context) (*models.AlertChannel, error) {
	req := models.AlertChannel{}
	if err := c.BindJSON(&req); err != nil {
		return nil, err
	}
	tenantID := c.Param("tenant_id")
	if tenantID != "_all" {
		t, _ := strconv.Atoi(tenantID)
		if t == 0 {
			return nil, fmt.Errorf("tenant id not valid")
		}
		tmp := uint(t)
		h.SetExtraAuditData(c, models.ResTenant, tmp)
		req.TenantID = &tmp
	}
	if req.ChannelConfig.ChannelIf == nil {
		return nil, fmt.Errorf("channel config can't be null")
	}
	if err := req.ChannelConfig.ChannelIf.Check(); err != nil {
		return nil, err
	}
	return &req, nil
}

// ListChannels 告警渠道列表
//	@Tags			Observability
//	@Summary		告警渠道列表
//	@Description	告警渠道列表
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id	path		string												true	"租户id, 所有租户为_all"
//	@Param			search		query		string												false	"search in (name)"
//	@Param			page		query		int													false	"page"
//	@Param			size		query		int													false	"size"
//	@Success		200			{object}	handlers.ResponseStruct{Data=[]models.AlertChannel}	"resp"
//	@Router			/v1/observability/tenant/{tenant_id}/channels [get]
//	@Security		JWT
func (h *ObservabilityHandler) ListChannels(c *gin.Context) {
	list := []models.AlertChannel{}
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:        "AlertChannel",
		SearchFields: []string{"name"},
	}
	tenantID := c.Param("tenant_id")
	if tenantID != "_all" {
		cond.Where = append(cond.Where, handlers.Args("tenant_id is null or tenant_id = ?", tenantID))
	}
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, handlers.Page(total, list, page, size))
}

// GetChannel 渠道列表详情
//	@Tags			Observability
//	@Summary		渠道列表详情
//	@Description	渠道列表详情
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id	path		string												true	"租户id, 所有租户为_all"
//	@Param			channel_id	path		string												true	"告警渠道id"
//	@Success		200			{object}	handlers.ResponseStruct{Data=models.AlertChannel}	"resp"
//	@Router			/v1/observability/tenant/{tenant_id}/channels/{channel_id} [get]
//	@Security		JWT
func (h *ObservabilityHandler) GetChannel(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	query := h.GetDB().WithContext(c.Request.Context())
	if tenantID != "_all" {
		query.Where("tenant_id = ? or tenant_id is null", tenantID)
	}
	ret := models.AlertChannel{}
	if err := query.First(&ret, "id = ?", c.Param("channel_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, ret)
}

// CreateChannel 创建告警渠道
//	@Tags			Observability
//	@Summary		创建告警渠道
//	@Description	创建告警渠道
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id	path		string									true	"租户id, 所有租户为_all"
//	@Param			form		body		models.AlertChannel						true	"body"
//	@Success		200			{object}	handlers.ResponseStruct{Data=string}	"resp"
//	@Router			/v1/observability/tenant/{tenant_id}/channels [post]
//	@Security		JWT
func (h *ObservabilityHandler) CreateChannel(c *gin.Context) {
	req, err := h.getChannelReq(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "创建", "告警渠道", req.Name)
	if req.TenantID == nil {
		if c.Param("tenant_id") != "_all" {
			handlers.NotOK(c, fmt.Errorf("你不能创建系统级告警渠道"))
			return
		}
	} else {
		h.SetExtraAuditData(c, models.ResTenant, *req.TenantID)
	}

	if err := h.GetDB().WithContext(c.Request.Context()).Create(req).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

// UpdateChannel 更新告警渠道
//	@Tags			Observability
//	@Summary		更新告警渠道
//	@Description	更新告警渠道
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id	path		string											true	"租户id, 所有租户为_all"
//	@Param			channel_id	path		string											true	"告警渠道id"
//	@Param			form		body		models.AlertChannel								true	"body"
//	@Success		200			{object}	handlers.ResponseStruct{Data=map[string]bool}	"告警规则-更新状态的map"
//	@Router			/v1/observability/tenant/{tenant_id}/channels/{channel_id} [put]
//	@Security		JWT
func (h *ObservabilityHandler) UpdateChannel(c *gin.Context) {
	req, err := h.getChannelReq(c)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "更新", "告警渠道", req.Name)
	if req.TenantID == nil {
		if c.Param("tenant_id") != "_all" {
			handlers.NotOK(c, fmt.Errorf("你不能更新系统级告警渠道"))
			return
		}
	} else {
		h.SetExtraAuditData(c, models.ResTenant, *req.TenantID)
	}

	// find associated alert rules
	alertrules := []*models.AlertRule{}
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Updates(req).Error; err != nil {
			return err
		}

		alertruleIDs := []uint{}
		if err := tx.Model(&models.AlertReceiver{}).Distinct("alert_rule_id").Where("alert_channel_id = ?", req.ID).Scan(&alertruleIDs).Error; err != nil {
			return err
		}
		return tx.Preload("Receivers.AlertChannel").Where("id in ?", alertruleIDs).Find(&alertrules).Error
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	status, _ := h.syncAlertRulesWithTimeout(ctx, alertrules, 5*time.Second)
	handlers.OK(c, status)
}

// DeleteChannel 删除告警渠道
//	@Tags			Observability
//	@Summary		删除告警渠道
//	@Description	删除告警渠道
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id	path		string									true	"租户id, 所有租户为_all"
//	@Param			channel_id	path		string									true	"告警渠道id"
//	@Param			form		body		models.AlertChannel						true	"body"
//	@Success		200			{object}	handlers.ResponseStruct{Data=string}	"resp"
//	@Router			/v1/observability/tenant/{tenant_id}/channels/{channel_id} [delete]
//	@Security		JWT
func (h *ObservabilityHandler) DeleteChannel(c *gin.Context) {
	ch := &models.AlertChannel{}
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).First(ch, "id = ?", c.Param("channel_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "删除", "告警渠道", ch.Name)
	if ch.TenantID == nil {
		if c.Param("tenant_id") != "_all" {
			handlers.NotOK(c, fmt.Errorf("你不能删除系统级告警渠道"))
			return
		}
	} else {
		h.SetExtraAuditData(c, models.ResTenant, *ch.TenantID)
	}

	receivers := []models.AlertReceiver{}
	if err := h.GetDB().WithContext(ctx).Preload("AlertRule").Find(&receivers, "alert_channel_id = ?", c.Param("channel_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if len(receivers) > 0 {
		tmp := make([]string, len(receivers))
		for i, v := range receivers {
			tmp[i] = v.AlertRule.FullName()
		}
		handlers.NotOK(c, fmt.Errorf("该告警渠道正在被告警规则: [%s] 使用", strings.Join(tmp, ",")))
		return
	}
	if err := h.GetDB().WithContext(ctx).Delete(ch).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}

// TestChannel 测试告警渠道
//	@Tags			Observability
//	@Summary		测试告警渠道
//	@Description	测试告警渠道
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id	path		string									true	"租户id, 所有租户为_all"
//	@Param			channel_id	path		string									true	"告警渠道id"
//	@Success		200			{object}	handlers.ResponseStruct{Data=string}	"resp"
//	@Router			/v1/observability/tenant/{tenant_id}/channels/{channel_id}/test [post]
//	@Security		JWT
func (h *ObservabilityHandler) TestChannel(c *gin.Context) {
	ch := &models.AlertChannel{}
	if err := h.GetDB().WithContext(c.Request.Context()).First(ch, "id = ?", c.Param("channel_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	now := time.Now()
	alertObj := prometheus.WebhookAlert{
		Receiver: ch.ReceiverName(),
		Status:   "firing",
		Alerts: []prometheus.Alert{
			{
				Status: "firing",
				Labels: map[string]string{
					prometheus.AlertNameLabel:      "kubegems-test-alert",
					prometheus.SeverityLabel:       prometheus.SeverityError,
					prometheus.AlertClusterKey:     "kubegems",
					prometheus.AlertNamespaceLabel: "kubegems-test-namespace",
				},
				Annotations: map[string]string{
					prometheus.MessageAnnotationsKey: "kubegems test alert message",
					prometheus.ValueAnnotationKey:    "0",
				},
				StartsAt: &now,
			},
		},
	}
	if err := ch.ChannelConfig.ChannelIf.Test(alertObj); err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, "ok")
}
