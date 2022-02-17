package models

import (
	"encoding/json"
	"time"
)

const (
	ResUser = "user"
)

// User 用户表
type User struct {
	ID uint `gorm:"primarykey"`
	// 用户名
	Username string `gorm:"type:varchar(50);uniqueIndex" binding:"required"`
	// 邮箱
	Email string `gorm:"type:varchar(50)" binding:"required"`
	// 电话
	Phone    string `gorm:"type:varchar(255)" binding:"required"`
	Password string `gorm:"type:varchar(255)" json:"-"`
	// 是否激活
	IsActive *bool `sql:"DEFAULT:true"`
	// 加入时间
	CreatedAt *time.Time `sql:"DEFAULT:'current_timestamp'"`
	// 最后登录时间
	LastLoginAt *time.Time `sql:"DEFAULT:'current_timestamp'"`

	Tenants      []*Tenant `gorm:"many2many:tenant_user_rels;"`
	SystemRole   *SystemRole
	SystemRoleID uint

	// 角色，不同关联对象下表示的角色不同, 用来做join查询的时候处理角色字段的(请勿删除)
	Role string `sql:"-" json:",omitempty"`
}

type UserSel struct {
	ID       uint
	Username string
	Email    string
}

// implement redis
func (u *User) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

func (u *User) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &u)
}
