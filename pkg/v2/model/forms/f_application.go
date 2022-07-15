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

// +genform object:Application
type ApplicationCommon struct {
	BaseForm
	ID        uint       `json:"id,omitempty"`
	Name      string     `json:"name,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// +genform object:Application
type ApplicationDetail struct {
	BaseForm
	ID            uint               `json:"id,omitempty"`
	Name          string             `json:"name,omitempty"`
	CreatedAt     *time.Time         `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time         `json:"updatedAt,omitempty"`
	Environment   *EnvironmentCommon `json:"environment,omitempty"`
	EnvironmentID *uint              `json:"environmentID,omitempty"`
	Project       *ProjectCommon     `json:"project,omitempty"`
	ProjectID     uint               `json:"projectID,omitempty"`
	Manifest      datatypes.JSON     `json:"manifest,omitempty"` // 应用manifest
	Remark        string             `json:"remark,omitempty"`   // 备注
	Kind          string             `json:"kind,omitempty"`     // 类型
	Enabled       bool               `json:"enabled,omitempty"`  // 激活状态
	Images        datatypes.JSON     `json:"images,omitempty"`   // 镜像,逗号分割
	Labels        datatypes.JSON     `json:"labels,omitempty"`   // Label
	Creator       string             `json:"creator,omitempty"`  // 创建人
}
