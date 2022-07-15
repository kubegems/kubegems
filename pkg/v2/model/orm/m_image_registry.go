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
)

// +gen type:object pkcolume:id pkfield:ID preloads:Project
type Registry struct {
	ID         uint   `gorm:"primarykey"`
	Name       string `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_registry;"`
	Address    string `gorm:"type:varchar(512)"`
	Username   string `gorm:"type:varchar(50)"`
	Password   string `gorm:"type:varchar(512)"`
	Creator    *User
	UpdateTime *time.Time
	CreatorID  uint
	Project    *Project `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ProjectID  uint     `grom:"uniqueIndex:uniq_idx_project_registry;"`
	IsDefault  bool
}
