package orm

import (
	"time"

	"gorm.io/datatypes"
)

// +gen type:object pkcolume:id pkfield:ID preloads:Tenant,Createor,Tenant,Environments
type Project struct {
	ID            uint       `gorm:"primarykey"`
	Name          string     `gorm:"type:varchar(50);uniqueIndex:uniq_idx_tenant_project_name"`
	CreatedAt     *time.Time `sql:"DEFAULT:'current_timestamp'"`
	ProjectAlias  string     `gorm:"type:varchar(50)"`
	Remark        string
	ResourceQuota datatypes.JSON
	Applications  []*Application
	Environments  []*Environment
	Users         []*User `gorm:"many2many:project_user_rels;"`
	Tenant        *Tenant `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	TenantID      uint    `gorm:"uniqueIndex:uniq_idx_tenant_project_name"`
}

// +gen type:objectrel pkcolume:id pkfield:ID preloads:User,Project
type ProjectUserRel struct {
	ID        uint     `gorm:"primarykey"`
	User      *User    `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Project   *Project `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	UserID    uint     `gorm:"uniqueIndex:uniq_idx_project_user_rel" binding:"required"`
	ProjectID uint     `gorm:"uniqueIndex:uniq_idx_project_user_rel" binding:"required"`
	Role      string   `gorm:"type:varchar(30)" binding:"required,eq=admin|eq=test|eq=dev|eq=ops"`
}
