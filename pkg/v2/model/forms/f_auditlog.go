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
	"gorm.io/gorm"
)

// +genform object:AuditLog
type AuditLogCommon struct {
	BaseForm
	ID        uint           `json:"id,omitempty"`
	CreatedAt *time.Time     `json:"createdAt,omitempty"`
	UpdatedAt *time.Time     `json:"updatedAt,omitempty"`
	DeletedAt gorm.DeletedAt `json:"deletedAt,omitempty"`
	Username  string         `json:"username,omitempty"`
	Tenant    string         `json:"tenant,omitempty"`
	Module    string         `json:"module,omitempty"`
	Name      string         `json:"name,omitempty"`
	Action    string         `json:"action,omitempty"`
	Success   bool           `json:"success,omitempty"`
	ClientIP  string         `json:"clientIP,omitempty"`
	Labels    datatypes.JSON `json:"labels,omitempty"`
	RawData   datatypes.JSON `json:"rawData,omitempty"`
}
