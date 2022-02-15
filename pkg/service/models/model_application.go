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
