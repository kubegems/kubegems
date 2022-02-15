package orm

import (
	"time"

	"gorm.io/datatypes"
)

// Application 应用表
// +gen type:object pkcolume:id pkfield:ID preloads:SystemRole
type Application struct {
	ID              uint           `gorm:"primarykey"`
	ApplicationName string         `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_applicationname;<-:create"` // 应用名字
	CreatedAt       *time.Time     `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt       *time.Time     // 创建时间
	Environment     *Environment   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // 关联的环境
	EnvironmentID   *uint          `gorm:"uniqueIndex:uniq_idx_project_applicationname;"` // 关联的环境
	Project         *Project       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // 所属项
	ProjectID       uint           `gorm:"uniqueIndex:uniq_idx_project_applicationname"`  // 所属项目ID
	Manifest        datatypes.JSON // 应用manifest
	Remark          string         // 备注
	Kind            string         // 类型
	Enabled         bool           // 激活状态
	Images          datatypes.JSON // 镜像,逗号分割
	Labels          datatypes.JSON // Label
	Creator         string         // 创建人
}
