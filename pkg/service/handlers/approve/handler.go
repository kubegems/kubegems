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
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/i18n"
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
					Title:        i18n.Sprintf(c, "user %s applied to adjust the ResourceQuota of tenant %s in cluster %s", v.TenantResourceQuotaApply.Username, v.Tenant.TenantName, v.Cluster.ClusterName),
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

// Approve 批准集群资源配额申请
// @Tags        Approve
// @Summary     批准集群资源配额申请
// @Description 批准集群资源配额申请
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
		handlers.NotOK(c, i18n.Errorf(c, "the tenant has no enough resources in the current cluster"))
		return
	}

	if quota.TenantResourceQuotaApply == nil {
		handlers.NotOK(c, i18n.Errorf(c, "the tenant has no resource application approval in the current cluster"))
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

	action := i18n.Sprintf(context.TODO(), "passed")
	module := i18n.Sprintf(context.TODO(), "cluster resource quota adjustment application")
	h.SetAuditData(c, action, module, quota.Tenant.TenantName+"/"+quota.Cluster.ClusterName)

	handlers.OK(c, quota)
}

// Approve 拒绝集群资源配额申请审批拒绝
// @Tags        Approve
// @Summary     拒绝集群资源配额申请审批拒绝
// @Description 拒绝集群资源配额申请审批拒绝
// @Accept      json
// @Produce     json
// @Param       id  path     uint                                 true "id"
// @Success     200 {object} handlers.ResponseStruct{Data=string} ""
// @Router      /v1/approve/{id}/reject [post]
// @Security    JWT
func (h *ApproveHandler) Reject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handlers.NotOK(c, i18n.Errorf(c, "there is no cluster resource quota adjustment approval"))
		return
	}
	quota := models.TenantResourceQuota{ID: uint(id)}
	if err := h.GetDB().Preload("TenantResourceQuotaApply").Preload("Tenant").Preload("Cluster").First(&quota).Error; err != nil {
		handlers.NotOK(c, fmt.Errorf("the tenant has no resource quota available in the current cluster"))
		return
	}

	if quota.TenantResourceQuotaApply == nil {
		handlers.NotOK(c, fmt.Errorf("the tenant has no resource quota approval in the current cluster"))
		return
	}

	if err := h.GetDB().Delete(quota.TenantResourceQuotaApply).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	targetUser := models.User{}
	h.GetDB().Where("username = ?", quota.TenantResourceQuotaApply.Username).First(&targetUser)

	action := i18n.Sprintf(context.TODO(), "rejected")
	module := i18n.Sprintf(context.TODO(), "cluster resource quota adjustment application")
	h.SetAuditData(c, action, module, quota.Tenant.TenantName+"/"+quota.Cluster.ClusterName)

	handlers.OK(c, quota)
}
