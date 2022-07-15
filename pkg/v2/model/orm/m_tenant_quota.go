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

package orm

import (
	"time"

	"gorm.io/datatypes"
)

const (
	TenantRoleAdmin    = "admin"
	TenantRoleOrdinary = "ordinary"
	ResTenant          = "tenant"
)

// TenantResourceQuota 租户集群资源限制表(限制一个租户在一个集群的资源使用量)
// +gen type:object pkcolume:id pkfield:ID preloads:Tenant,Cluster,TenantResourceQuotaApply
type TenantResourceQuota struct {
	ID        uint
	Content   datatypes.JSON
	TenantID  uint     `gorm:"uniqueIndex:uniq_tenant_cluster"`
	ClusterID uint     `gorm:"uniqueIndex:uniq_tenant_cluster"`
	Tenant    *Tenant  `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Cluster   *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
}

// TenantResourceQuotaApply  租户集群资源申请表
// +gen type:object pkcolume:id pkfield:ID
type TenantResourceQuotaApply struct {
	ID        uint
	Content   datatypes.JSON
	Status    string     `gorm:"type:varchar(30);"`
	CreateAt  *time.Time `sql:"DEFAULT:'current_timestamp'"`
	TenantID  uint       `gorm:"uniqueIndex:uniq_tenant_cluster_quota_apply"`
	ClusterID uint       `gorm:"uniqueIndex:uniq_tenant_cluster_quota_apply"`
	Tenant    *Tenant    `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Cluster   *Cluster   `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Creator   *User
	CreatorID uint
}
