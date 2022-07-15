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

package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:Project
type ProjectCommon struct {
	BaseForm
	ID           uint       `json:"id"`
	CreatedAt    *time.Time `json:"createAt"`
	Name         string     `json:"projectName"`
	ProjectAlias string     `json:"projectAlias"`
	Remark       string     `json:"remark"`
}

// +genform object:Project
type ProjectDetail struct {
	BaseForm
	ID            uint                 `json:"id"`
	CreatedAt     *time.Time           `json:"createAt"`
	Name          string               `json:"projectName"`
	ProjectAlias  string               `json:"projectAlias"`
	Remark        string               `json:"remark"`
	ResourceQuota datatypes.JSON       `json:"resourecQuota"`
	Applications  []*ApplicationCommon `json:"applications,omitempty"`
	Environments  []*EnvironmentCommon `json:"environments,omitempty"`
	Users         []*UserCommon        `json:"users,omitempty"`
	Tenant        *TenantCommon        `json:"tenant,omitempty"`
	TenantID      uint                 `json:"tenantId,omitempty"`
}

// +genform object:ProjectUserRel
type ProjectUserRelCommon struct {
	BaseForm
	ID        uint           `json:"id"`
	User      *UserCommon    `json:"user,omitempty"`
	Project   *ProjectCommon `json:"project,omitempty"`
	UserID    uint           `json:"userId"`
	ProjectID uint           `json:"projectId"`
	Role      string         `json:"role"`
}

type ProjectCreateForm struct {
	BaseForm
	Name          string `json:"name" validate:"required"`
	Remark        string `json:"remark" validate:"required"`
	ResourceQuota string `json:"quota" validate:"json"`
}
