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

package models

import (
	"time"

	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
)

const (
	TenantRoleAdmin    = "admin"
	TenantRoleOrdinary = "ordinary"
	ResTenant          = "tenant"
)

// Tenant 租户表
type Tenant struct {
	ID uint `gorm:"primarykey"`
	// 租户名字
	TenantName string `gorm:"type:varchar(50);uniqueIndex"`
	// 备注
	Remark string
	// 是否激活
	IsActive  bool
	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`

	ResourceQuotas []*TenantResourceQuota
	Users          []*User `gorm:"many2many:tenant_user_rels;"`
	Projects       []*Project

	AllocatedResourcequota v1.ResourceList `gorm:"-"`
}

// TenantUserRels 租户-用户-关系表
// 租户id-用户id-类型 唯一索引
type TenantUserRels struct {
	ID     uint    `gorm:"primarykey"`
	Tenant *Tenant `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 租户ID
	TenantID uint  `gorm:"uniqueIndex:uniq_idx_tenant_user_rel"`
	User     *User `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 用户ID
	UserID uint `gorm:"uniqueIndex:uniq_idx_tenant_user_rel"`

	// 租户级角色(管理员admin, 普通用户ordinary)
	Role string `gorm:"type:varchar(30)" binding:"required"`
}

type TenantResourceQuota struct {
	ID      uint
	Content datatypes.JSON

	TenantID                   uint                      `gorm:"uniqueIndex:uniq_tenant_cluster" binding:"required"`
	ClusterID                  uint                      `gorm:"uniqueIndex:uniq_tenant_cluster" binding:"required"`
	Tenant                     *Tenant                   `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Cluster                    *Cluster                  `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	TenantResourceQuotaApply   *TenantResourceQuotaApply `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:SET NULL;"`
	TenantResourceQuotaApplyID *uint
}

const (
	QuotaStatusApproved = "approved"
	QuotaStatusRejected = "rejected"
	QuotaStatusPending  = "pending"
)

// TenantResourceQuotaApply 集群资源申请
type TenantResourceQuotaApply struct {
	ID        uint
	Content   datatypes.JSON
	Status    string    `gorm:"type:varchar(30);"`
	Username  string    `gorm:"type:varchar(255);"`
	UpdatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`
}
