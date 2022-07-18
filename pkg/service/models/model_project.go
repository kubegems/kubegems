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
	ProjectRoleAdmin = "admin"
	ProjectRoleDev   = "dev"
	ProjectRoleTest  = "test"
	ProjectRoleOps   = "ops"

	ResProject = "project"
)

// Project 项目表
type Project struct {
	ID uint `gorm:"primarykey"`
	// 创建时间
	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`
	// 项目名字
	ProjectName string `gorm:"type:varchar(50);uniqueIndex:uniq_idx_tenant_project_name"`
	// 项目别名
	ProjectAlias string `gorm:"type:varchar(50)"`
	// 项目备注
	Remark string
	// 项目资源限制
	ResourceQuota datatypes.JSON

	Applications []*Application
	Environments []*Environment
	Registries   []*Registry
	Users        []*User `gorm:"many2many:project_user_rels;"`
	Tenant       *Tenant `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 所属的租户ID
	TenantID uint `gorm:"uniqueIndex:uniq_idx_tenant_project_name"`
}

// ProjectUserRels
type ProjectUserRels struct {
	ID      uint     `gorm:"primarykey"`
	User    *User    `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Project *Project `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 用户ID
	UserID uint `gorm:"uniqueIndex:uniq_idx_project_user_rel" binding:"required"`
	// ProjectID
	ProjectID uint `gorm:"uniqueIndex:uniq_idx_project_user_rel" binding:"required"`

	// 项目级角色(管理员admin, 开发dev, 测试test, 运维ops)
	Role string `gorm:"type:varchar(30)" binding:"required,eq=admin|eq=test|eq=dev|eq=ops"`
}
