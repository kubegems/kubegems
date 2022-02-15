package orm

import "time"

// User 用户表
// +gen type:object pkcolume:id pkfield:ID preloads:SystemRole
type User struct {
	ID           uint       `gorm:"primarykey"`
	Username     string     `gorm:"type:varchar(50);uniqueIndex" binding:"required"`
	Email        string     `gorm:"type:varchar(50)" binding:"required"`
	Phone        string     `gorm:"type:varchar(255)" binding:"required" json:",omitempty"`
	Password     string     `gorm:"type:varchar(255)" json:"-"`
	Source       string     `gorm:"type:varchar(255)"`
	IsActive     *bool      `sql:"DEFAULT:true"`
	CreatedAt    *time.Time `sql:"DEFAULT:'current_timestamp'"`
	LastLoginAt  *time.Time `sql:"DEFAULT:'current_timestamp'"`
	Tenants      []*Tenant  `gorm:"many2many:tenant_user_rels;"`
	SystemRole   *SystemRole
	SystemRoleID uint

	Role string `sql:"-" json:",omitempty"`
}
