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
	"gorm.io/gorm"
)

// +gen type:object kind:auditlog pkcolume:id pkfield:ID
type AuditLog struct {
	ID        uint       `gorm:"primarykey"`
	Name      string     `gorm:"type:varchar(512)"`
	CreatedAt *time.Time `gorm:"index"`
	UpdatedAt *time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Username  string         `gorm:"type:varchar(50)"`
	Tenant    string         `gorm:"type:varchar(50)"`
	Module    string         `gorm:"type:varchar(512)"`
	Action    string         `gorm:"type:varchar(255)"`
	Success   bool
	ClientIP  string `gorm:"type:varchar(255)"`
	Labels    datatypes.JSON
	RawData   datatypes.JSON
}
