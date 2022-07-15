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

// Application 应用表
type Application struct {
	ID              uint           `gorm:"primarykey"`
	ApplicationName string         `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_applicationname;<-:create"` // 应用名字
	Environment     *Environment   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`                           // 关联的环境
	EnvironmentID   *uint          `gorm:"uniqueIndex:uniq_idx_project_applicationname;"`                           // 关联的环境
	Project         *Project       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`                           // 所属项
	ProjectID       uint           `gorm:"uniqueIndex:uniq_idx_project_applicationname"`                            // 所属项目ID
	Remark          string         // 备注
	Kind            string         // 类型
	Images          datatypes.JSON // 镜像,逗号分割
	Labels          datatypes.JSON // Label
	Creator         string         // 创建人
	CreatedAt       time.Time      `sql:"DEFAULT:'current_timestamp'"` // 创建时间
}
