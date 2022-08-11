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

package approveHandler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	msgclient "kubegems.io/kubegems/pkg/msgbus/client"
	"kubegems.io/kubegems/pkg/service/handlers"
	tenanthandler "kubegems.io/kubegems/pkg/service/handlers/tenant"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/msgbus"
)

type Approve struct {
	msgbus.ResourceType
	ID          uint // 目前记录的是quota id
	Title       string
	Content     interface{}
	TenantID    uint   `json:",omitempty"`
	TenantName  string `json:",omitempty"`
	ClusterID   uint   `json:",omitempty"`
	ClusterName string `json:",omitempty"`
	CreatedAt   time.Time
	Status      string
}

type ApprovesList []Approve

func (a ApprovesList) Len() int           { return len(a) }
func (a ApprovesList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ApprovesList) Less(i, j int) bool { return a[i].CreatedAt.After(a[j].CreatedAt) } // 倒序

// ListApproves 获取待处理审批
// @Tags        Approve
// @Summary     获取待处理审批
// @Description 获取待处理审批
// @Accept      json
// @Produce     json
// @Success     200 {object} handlers.ResponseStruct{Data=ApprovesList} "ApprovesList"
// @Router      /v1/approve [get]
// @Security    JWT
func (h *ApproveHandler) ListApproves(c *gin.Context) {
	// 审批中的，查quota
	var quotas []models.TenantResourceQuota
	if err := h.GetDB().
		Preload("Tenant").
		Preload("Cluster").
		Preload("TenantResourceQuotaApply").
		Find(&quotas).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	ret := ApprovesList{}
	// 目前只给admin看
	u, _ := h.GetContextUser(c)
	if h.ModelCache().GetUserAuthority(u).IsSystemAdmin() {
		for _, v := range quotas {
			if v.TenantResourceQuotaApply != nil && v.TenantResourceQuotaApply.Status == models.QuotaStatusPending {
				ret = append(ret, Approve{
					ResourceType: msgbus.TenantResourceQuota,
					ID:           v.ID,
					Title:        fmt.Sprintf("用户%s申请调整租户%s在集群%s的资源", v.TenantResourceQuotaApply.Username, v.Tenant.TenantName, v.Cluster.ClusterName),
					Content:      v.TenantResourceQuotaApply.Content,
					TenantID:     v.TenantID,
					TenantName:   v.Tenant.TenantName,
					ClusterID:    v.ClusterID,
					ClusterName:  v.Cluster.ClusterName,
					CreatedAt:    v.TenantResourceQuotaApply.UpdatedAt,
					Status:       models.QuotaStatusPending,
				})
			}
		}
		sort.Sort(ret)
	}

	handlers.OK(c, ret)
}

// Approve 审批通过
// @Tags        Approve
// @Summary     审批通过
// @Description 审批通过
// @Accept      json
// @Produce     json
// @Param       id    path     uint                                 true "tenant resource quota id"
// @Param       param body     Approve                              true "通过的内容"
// @Success     200   {object} handlers.ResponseStruct{Data=string} ""
// @Router      /v1/approve/{id}/pass [post]
// @Security    JWT
func (h *ApproveHandler) Pass(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	quota := models.TenantResourceQuota{ID: uint(id)}
	if err := h.GetDB().Preload("TenantResourceQuotaApply").Preload("Tenant").Preload("Cluster").First(&quota).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("租户在当前集群不存在可以使用资源"))
		return
	}

	if quota.TenantResourceQuotaApply == nil {
		handlers.NotOK(c, fmt.Errorf("租户在当前集群没有资源申请审批"))
		return
	}

	req := Approve{}
	if err := c.BindJSON(&req); err != nil {
		handlers.NotOK(c, err)
		return
	}

	content, err := json.Marshal(req.Content)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	targetUser := models.User{}
	h.GetDB().Where("username = ?", quota.TenantResourceQuotaApply.Username).First(&targetUser)

	// 应用新的resource quota
	ctx := c.Request.Context()
	quota.Content = content
	if e := h.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&quota).Error; err != nil {
			return err
		}
		return tenanthandler.AfterTenantResourceQuotaSave(ctx, h.BaseHandler, tx, &quota)
	}); e != nil {
		handlers.NotOK(c, err)
		return
	}

	// 外键是SET NULL，直接删除记录即可
	if err := h.GetDB().Delete(quota.TenantResourceQuotaApply).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "通过", "集群资源申请", quota.Tenant.TenantName+"/"+quota.Cluster.ClusterName)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.MessageType = msgbus.Approve
		msg.EventKind = msgbus.Update
		msg.ResourceType = msgbus.TenantResourceQuota
		msg.ResourceID = quota.ID
		msg.Detail = fmt.Sprintf("通过了用户%s在租户%s中发起对集群%s的资源调整审批", targetUser.Username, quota.Tenant.TenantName, quota.Cluster.ClusterName)
		msg.ToUsers.Append(h.GetDataBase().TenantAdmins(quota.TenantID)...).Append(targetUser.ID)
	})

	handlers.OK(c, quota)
}

// Approve 审批拒绝
// @Tags        Approve
// @Summary     审批拒绝
// @Description 审批拒绝
// @Accept      json
// @Produce     json
// @Param       id  path     uint                                 true "id"
// @Success     200 {object} handlers.ResponseStruct{Data=string} ""
// @Router      /v1/approve/{id}/reject [post]
// @Security    JWT
func (h *ApproveHandler) Reject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handlers.NotOK(c, fmt.Errorf("申请不存在"))
		return
	}
	quota := models.TenantResourceQuota{ID: uint(id)}
	if err := h.GetDB().Preload("TenantResourceQuotaApply").Preload("Tenant").Preload("Cluster").First(&quota).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("租户在当前集群不存在可以使用资源"))
		return
	}

	if quota.TenantResourceQuotaApply == nil {
		handlers.NotOK(c, fmt.Errorf("租户在当前集群没有资源申请审批"))
		return
	}

	// 外键是SET NULL，直接删除记录即可
	if err := h.GetDB().Delete(quota.TenantResourceQuotaApply).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	targetUser := models.User{}
	h.GetDB().Where("username = ?", quota.TenantResourceQuotaApply.Username).First(&targetUser)

	h.SetAuditData(c, "拒绝", "集群资源申请", quota.Tenant.TenantName+"/"+quota.Cluster.ClusterName)

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.MessageType = msgbus.Approve
		msg.EventKind = msgbus.Update
		msg.ResourceType = msgbus.TenantResourceQuota
		msg.ResourceID = quota.ID
		msg.Detail = fmt.Sprintf("拒绝了用户%s在租户%s中发起对集群%s的资源调整审批", targetUser.Username, quota.Tenant.TenantName, quota.Cluster.ClusterName)
		msg.ToUsers.Append(h.GetDataBase().TenantAdmins(quota.TenantID)...).Append(targetUser.ID)
	})

	handlers.OK(c, quota)
}
