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
