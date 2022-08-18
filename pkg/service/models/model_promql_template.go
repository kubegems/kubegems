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

	"kubegems.io/kubegems/pkg/utils/gormdatatypes"
)

// PromqlTplScope
type PromqlTplScope struct {
	ID         uint   `gorm:"primarykey" json:"id"`
	Name       string `gorm:"type:varchar(50)" binding:"required" json:"name"`
	ShowName   string `gorm:"type:varchar(50)" json:"showName"`
	Namespaced bool   `json:"namespaced"`

	Resources []*PromqlTplResource `json:"resources,omitempty" gorm:"foreignKey:ScopeID"`

	CreatedAt *time.Time `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// PromqlTplScope
type PromqlTplResource struct {
	ID       uint   `gorm:"primarykey" json:"id"`
	Name     string `gorm:"type:varchar(50)" binding:"required" json:"name"`
	ShowName string `gorm:"type:varchar(50)" json:"showName"`

	ScopeID *uint           `json:"scopeID"`
	Scope   *PromqlTplScope `gorm:"foreignKey:ScopeID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"scope,omitempty"`

	Rules []*PromqlTplRule `json:"rules,omitempty" gorm:"foreignKey:ResourceID"`

	CreatedAt *time.Time `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// PromqlTplScope
type PromqlTplRule struct {
	ID          uint                    `gorm:"primarykey" json:"id"`
	Name        string                  `gorm:"type:varchar(50)" binding:"required" json:"name"`
	ShowName    string                  `gorm:"type:varchar(50)" json:"showName"`
	Description string                  `json:"description"`
	Expr        string                  `json:"expr"` // promql expr
	Unit        string                  `gorm:"type:varchar(50)" json:"unit"`
	Labels      gormdatatypes.JSONSlice `json:"labels"`

	ResourceID *uint              `json:"resourceID"`
	Resource   *PromqlTplResource `gorm:"foreignKey:ResourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"resource,omitempty"`

	TenantID *uint   `json:"tenantID"` // 若为null，则表示系统预置
	Tenant   *Tenant `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;" json:"tenant,omitempty"`

	CreatedAt *time.Time `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}
