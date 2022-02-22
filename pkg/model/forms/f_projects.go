package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:Project
type ProjectCommon struct {
	BaseForm
	ID           uint       `json:"id"`
	CreatedAt    *time.Time `json:"createAt"`
	ProjectName  string     `json:"projectName"`
	ProjectAlias string     `json:"projectAlias"`
	Remark       string     `json:"remark"`
}

// +genform object:Project
type ProjectDetail struct {
	BaseForm
	ID            uint                 `json:"id"`
	CreatedAt     *time.Time           `json:"createAt"`
	ProjectName   string               `json:"projectName"`
	ProjectAlias  string               `json:"projectAlias"`
	Remark        string               `json:"remark"`
	ResourceQuota datatypes.JSON       `json:"resourecQuota"`
	Applications  []*ApplicationCommon `json:"applications,omitempty"`
	Environments  []*EnvironmentCommon `json:"environments,omitempty"`
	Users         []*UserCommon        `json:"users,omitempty"`
	Tenant        *TenantCommon        `json:"tenant,omitempty"`
	TenantID      uint                 `json:"tenantId,omitempty"`
}

// +genform object:ProjectUserRel
type ProjectUserRelCommon struct {
	BaseForm
	ID        uint           `json:"id"`
	User      *UserCommon    `json:"user,omitempty"`
	Project   *ProjectCommon `json:"project,omitempty"`
	UserID    uint           `json:"userId"`
	ProjectID uint           `json:"projectId"`
	Role      string         `json:"role"`
}

type ProjectCreateForm struct {
	BaseForm
	Name          string `json:"name" validate:"required"`
	Remark        string `json:"remark" validate:"required"`
	ResourceQuota string `json:"quota" validate:"json"`
}
