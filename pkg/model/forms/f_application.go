package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:Application
type ApplicationCommon struct {
	BaseForm
	ID              uint
	ApplicationName string
	CreatedAt       *time.Time
	UpdatedAt       *time.Time // 创建时间
}

// +genform object:Application
type ApplicationDetail struct {
	BaseForm
	ID              uint
	ApplicationName string
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
	Environment     *EnvironmentCommon
	EnvironmentID   *uint
	Project         *ProjectCommon
	ProjectID       uint
	Manifest        datatypes.JSON // 应用manifest
	Remark          string         // 备注
	Kind            string         // 类型
	Enabled         bool           // 激活状态
	Images          datatypes.JSON // 镜像,逗号分割
	Labels          datatypes.JSON // Label
	Creator         string         // 创建人
}
