package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:Project
type ProjectCommon struct {
	BaseForm
	ID uint
	// 创建时间
	CreatedAt *time.Time
	// 项目名字
	ProjectName string
	// 项目别名
	ProjectAlias string
	// 项目备注
	Remark string
	// 项目资源限制
}

// +genform object:Project
type ProjectDetail struct {
	BaseForm
	ID uint
	// 创建时间
	CreatedAt *time.Time
	// 项目名字
	ProjectName string
	// 项目别名
	ProjectAlias string
	// 项目备注
	Remark        string
	ResourceQuota datatypes.JSON
	Applications  []*ApplicationCommon
	Environments  []*EnvironmentCommon
	Users         []*UserCommon
	Tenant        *TenantCommon
	// 所属的租户ID
	TenantID uint
}

// +genform object:ProjectUserRel
type ProjectUserRelCommon struct {
	BaseForm
	ID        uint
	User      *UserCommon
	Project   *ProjectCommon
	UserID    uint
	ProjectID uint
	Role      string
}
