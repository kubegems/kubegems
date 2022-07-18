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

// +gen type:object pkcolume:id pkfield:ID preloads:Environment
type Application struct {
	ID            uint       `gorm:"primarykey"`
	Name          string     `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_application;<-:create"`
	CreatedAt     *time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt     *time.Time
	Environment   *Environment `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	EnvironmentID *uint        `gorm:"uniqueIndex:uniq_idx_project_applicationname;"`
	Project       *Project     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ProjectID     uint         `gorm:"uniqueIndex:uniq_idx_project_applicationname"`
	Manifest      datatypes.JSON
	Remark        string
	Kind          string
	Enabled       bool
	Images        datatypes.JSON
	Labels        datatypes.JSON
	Creator       string
}
