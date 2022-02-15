package orm

import (
	"time"
)

// Tenant 租户
// +gen type:object pkcolume:id pkfield:ID preloads:Users,Projects
type Tenant struct {
	ID         uint   `gorm:"primarykey"`
	TenantName string `gorm:"type:varchar(50);uniqueIndex"`
	Remark     string
	IsActive   bool
	CreatedAt  *time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt  *time.Time `sql:"DEFAULT:'current_timestamp'"`

	ResourceQuotas []*TenantResourceQuota
	Users          []*User `gorm:"many2many:tenant_user_rels;"`
	Projects       []*Project
}

// TenantUserRel 租户用户关联关系表
// +gen type:objectrel pkcolume:id pkfield:ID preloads:User,Tenant leftfield:Tenant rightfield:User
type TenantUserRel struct {
	ID       uint    `gorm:"primarykey"`
	Tenant   *Tenant `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	TenantID uint    `gorm:"uniqueIndex:uniq_idx_tenant_user_rel"`
	User     *User   `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	UserID   uint    `gorm:"uniqueIndex:uniq_idx_tenant_user_rel"`
	Role     string  `gorm:"type:varchar(30)" binding:"required"`
}
