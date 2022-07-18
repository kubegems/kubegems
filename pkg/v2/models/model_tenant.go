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
)

const (
	TenantRoleAdmin    = "admin"
	TenantRoleOrdinary = "ordinary"
	ResTenant          = "tenant"

	TenantTableName     = "tenants"
	QuotaStatusApproved = "approved"
	QuotaStatusRejected = "rejected"
	QuotaStatusPending  = "pending"
)

/*
ALTER TABLE tenants RENAME COLUMN tenant_name TO name;
*/

type Tenant struct {
	ID             uint   `gorm:"primarykey"`
	Name           string `gorm:"type:varchar(50);uniqueIndex"`
	Remark         string
	IsActive       bool
	CreatedAt      time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt      time.Time `sql:"DEFAULT:'current_timestamp'"`
	ResourceQuotas []*TenantResourceQuota
	Users          []*User `gorm:"many2many:tenant_user_rels;"`
	Projects       []*Project
}

type TenantUserRels struct {
	ID       uint    `gorm:"primarykey"`
	Tenant   *Tenant `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	TenantID uint    `gorm:"uniqueIndex:uniq_idx_tenant_user_rel"`
	User     *User   `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	UserID   uint    `gorm:"uniqueIndex:uniq_idx_tenant_user_rel"`
	Role     string  `gorm:"type:varchar(30)" binding:"required"`
}

type TenantResourceQuota struct {
	ID        uint
	Content   datatypes.JSON
	TenantID  uint     `gorm:"uniqueIndex:uniq_tenant_cluster" binding:"required"`
	ClusterID uint     `gorm:"uniqueIndex:uniq_tenant_cluster" binding:"required"`
	Tenant    *Tenant  `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Cluster   *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
}

// TenantResourceQuotaApply 集群资源申请
type TenantResourceQuotaApply struct {
	ID        uint
	Content   datatypes.JSON
	Status    string    `gorm:"type:varchar(30);"`
	Username  string    `gorm:"type:varchar(255);"`
	UpdatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`
	CreateAt  time.Time `sql:"DEFAULT:'current_timestamp'"`
	Creator   *User
	CreatorID uint
	TenantID  uint     `gorm:"uniqueIndex:uniq_tenant_cluster_rq" binding:"required"`
	ClusterID uint     `gorm:"uniqueIndex:uniq_tenant_cluster_rq" binding:"required"`
	Tenant    *Tenant  `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Cluster   *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
}

// --- datas below

type TenantSimple struct {
	ID     uint   `json:"id,omitempty"`
	Name   string `validate:"required" json:"name,omitempty"`
	Remark string `validate:"required" json:"remark,omitempty"`
}

func (TenantSimple) TableName() string {
	return TenantTableName
}

type TenantCommon struct {
	ID        uint      `json:"id,omitempty"`
	Name      string    `json:"name,omitempty"`
	Remark    string    `json:"remark,omitempty"`
	IsActive  bool      `json:"isActive,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

func (TenantCommon) TableName() string {
	return TenantTableName
}
