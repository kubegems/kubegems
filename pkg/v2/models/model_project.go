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

/*
ALTER TABLE projects RENAME COLUMN project_name TO name
*/

const (
	ProjectRoleAdmin = "admin"
	ProjectRoleDev   = "dev"
	ProjectRoleTest  = "test"
	ProjectRoleOps   = "ops"

	ResProject       = "project"
	ProjectTableName = "projects"
)

type Project struct {
	ID            uint      `gorm:"primarykey"`
	CreatedAt     time.Time `sql:"DEFAULT:'current_timestamp'"`
	Name          string    `gorm:"type:varchar(50);uniqueIndex:uniq_idx_tenant_project_name"`
	ProjectAlias  string    `gorm:"type:varchar(50)"`
	Remark        string
	ResourceQuota datatypes.JSON
	Applications  []*Application
	Environments  []*Environment
	Registries    []*Registry
	Users         []*User `gorm:"many2many:project_user_rels;"`
	Tenant        *Tenant `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	TenantID      uint    `gorm:"uniqueIndex:uniq_idx_tenant_project_name"`
}

type ProjectUserRels struct {
	ID        uint     `gorm:"primarykey"`
	User      *User    `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Project   *Project `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	UserID    uint     `gorm:"uniqueIndex:uniq_idx_project_user_rel" binding:"required"`
	ProjectID uint     `gorm:"uniqueIndex:uniq_idx_project_user_rel" binding:"required"`
	Role      string   `gorm:"type:varchar(30)" binding:"required,eq=admin|eq=test|eq=dev|eq=ops"`
}

type ProjectCommon struct {
	ID            uint           `json:"id,omitempty"`
	CreatedAt     time.Time      `json:"createdAt,omitempty"`
	Name          string         `json:"name,omitempty"`
	ProjectAlias  string         `json:"projectAlias,omitempty"`
	Remark        string         `json:"remark,omitempty"`
	ResourceQuota datatypes.JSON `json:"resourceQuota,omitempty"`
}

func (ProjectCommon) TableName() string {
	return ProjectTableName
}
