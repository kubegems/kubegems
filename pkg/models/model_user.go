package models

import (
	"encoding/json"
	"time"
)

const (
	ResUser = "user"

	UserTableName = "users"
)

type User struct {
	ID           uint       `gorm:"primarykey"`
	Username     string     `gorm:"type:varchar(50);uniqueIndex" binding:"required"`
	Email        string     `gorm:"type:varchar(50)" binding:"required"`
	Phone        string     `gorm:"type:varchar(255)" binding:"required"`
	Password     string     `gorm:"type:varchar(255)" json:"-"`
	IsActive     *bool      `sql:"DEFAULT:true"`
	Source       string     `gorm:"type:varchar(255)" json:"-"`
	CreatedAt    *time.Time `sql:"DEFAULT:'current_timestamp'"`
	LastLoginAt  *time.Time `sql:"DEFAULT:'current_timestamp'"`
	Tenants      []*Tenant  `gorm:"many2many:tenant_user_rels;"`
	SystemRole   *SystemRole
	SystemRoleID uint
}

type UserSel struct {
	ID       uint
	Username string
	Email    string
}

// implement redis
func (u User) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

func (u *User) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &u)
}

type UserSimple struct {
	ID       uint   `json:"id,omitempty"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
}

type UserCommon struct {
	ID           uint        `json:"id,omitempty"`
	Username     string      `json:"username,omitempty"`
	Email        string      `json:"email,omitempty"`
	Phone        string      `json:"phone,omitempty"`
	IsActive     *bool       `json:"isActive,omitempty"`
	CreatedAt    *time.Time  `json:"createdAt,omitempty"`
	Source       string      `gorm:"type:varchar(255)" json:"-"`
	LastLoginAt  *time.Time  `json:"lastLoginAt,omitempty"`
	SystemRole   *SystemRole `json:"systemRole,omitempty"`
	SystemRoleID uint        `json:"systemRoleID,omitempty"`
}

func (u UserCommon) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

func (u *UserCommon) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &u)
}

func (UserSimple) TableName() string {
	return UserTableName
}

func (UserCommon) TableName() string {
	return UserTableName
}

func (u *UserCommon) GetID() uint {
	return u.ID
}
func (u *UserCommon) GetSystemRoleID() uint {
	return u.SystemRoleID
}
func (u *UserCommon) GetUsername() string {
	return u.Username
}
func (u *UserCommon) GetUserKind() string {
	return "inner"
}
func (u *UserCommon) GetEmail() string {
	return u.Email
}
func (u *UserCommon) GetSource() string {
	return u.Source
}
func (u *UserCommon) SetLastLogin(t *time.Time) {
	u.LastLoginAt = t
}
