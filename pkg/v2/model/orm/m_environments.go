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

// +gen type:object pkcolume:id pkfield:ID preloads:Cluster,Creator,Project,Applications,VirtualSpace
type Environment struct {
	ID           uint   `gorm:"primarykey"`
	Name         string `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_env;index:environment_uniq,unique"`
	Namespace    string `gorm:"type:varchar(50)"`
	Remark       string
	MetaType     string
	DeletePolicy string `sql:"DEFAULT:'delNamespace'"`

	Creator       *User
	Cluster       *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Project       *Project `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ResourceQuota datatypes.JSON
	LimitRange    datatypes.JSON
	ProjectID     uint `gorm:"uniqueIndex:uniq_idx_project_env"`
	ClusterID     uint
	CreatorID     uint
	Applications  []*Application `gorm:"many2many:application_environment_rels;"`
	Users         []*User        `gorm:"many2many:environment_user_rels;"`

	VirtualSpaceID *uint
	VirtualSpace   *VirtualSpace `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:SET NULL;"`
}

// +gen type:objectrel pkcolume:id pkfield:ID preloads:User,Environment
type EnvironmentUserRel struct {
	ID            uint `gorm:"primarykey"`
	User          *User
	Environment   *Environment `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	UserID        uint         `gorm:"uniqueIndex:uniq_idx_env_user_rel"`
	EnvironmentID uint         `gorm:"uniqueIndex:uniq_idx_env_user_rel"`
	// 环境级角色("reader", "operator")
	Role string `binding:"required,eq=reader|eq=operator"`
}

// +gen type:object pkcolume:id pkfield:ID
type EnvironmentResource struct {
	ID        uint       `gorm:"primarykey"`
	CreatedAt *time.Time `sql:"DEFAULT:'current_timestamp'"`

	Cluster     string
	Tenant      string
	Project     string
	Environment string

	MaxCPUUsageCore    float64
	MaxMemoryUsageByte float64
	MinCPUUsageCore    float64
	MinMemoryUsageByte float64
	AvgCPUUsageCore    float64
	AvgMemoryUsageByte float64
	NetworkReceiveByte float64
	NetworkSendByte    float64

	MaxPVCUsageByte float64
	MinPVCUsageByte float64
	AvgPVCUsageByte float64
}
