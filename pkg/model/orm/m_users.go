package orm

import "time"

// User 用户表
// +gen type:object pkcolume:id pkfield:ID preloads:SystemRole
type User struct {
	ID           uint       `gorm:"primarykey"`
	Name         string     `gorm:"type:varchar(50);uniqueIndex"`
	Email        string     `gorm:"type:varchar(50)"`
	Phone        string     `gorm:"type:varchar(255)"`
	Password     string     `gorm:"type:varchar(255)"`
	Source       string     `gorm:"type:varchar(255)"`
	IsActive     *bool      `sql:"DEFAULT:true"`
	CreatedAt    *time.Time `sql:"DEFAULT:'current_timestamp'"`
	LastLoginAt  *time.Time `sql:"DEFAULT:'current_timestamp'"`
	Tenants      []*Tenant  `gorm:"many2many:tenant_user_rels;"`
	SystemRole   *SystemRole
	SystemRoleID uint
	Role         string `sql:"-"`
}
