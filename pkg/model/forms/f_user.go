package forms

import (
	"encoding/json"
	"time"
)

// +genform object:User
type UserCommon struct {
	BaseForm
	ID       uint   `json:"id"`
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Role     string `json:"role,omitempty"`
}

// +genform object:User
type UserDetail struct {
	BaseForm
	ID           uint
	Username     string            `json:"username" validate:"required"`
	Email        string            `json:"email" validate:"required,email"`
	Phone        string            `json:"phone"`
	Source       string            `json:"source"`
	IsActive     *bool             `json:"is_active"`
	CreatedAt    *time.Time        `json:"created_at"`
	LastLoginAt  *time.Time        `json:"last_login_at"`
	SystemRole   *SystemRoleCommon `json:"system_role"`
	SystemRoleID uint              `json:"system_role_id"`

	Role string `json:"role,omitempty"`
}

// +genform object:User
type UserSetting struct {
	BaseForm
	ID           uint
	Username     string `binding:"required"`
	Phone        string `binding:"required" json:",omitempty"`
	Email        string
	Password     string
	IsActive     *bool
	SystemRole   *SystemRoleCommon
	SystemRoleID uint
}

// +genform object:User
type UserInternal struct {
	BaseForm
	ID           uint
	Username     string
	Password     string
	Email        string
	Role         string
	Phone        string
	Source       string
	IsActive     *bool
	CreatedAt    *time.Time
	LastLoginAt  *time.Time
	SystemRole   *SystemRoleCommon
	SystemRoleID uint
}

func (u *UserInternal) GetID() uint {
	return u.ID
}

func (u *UserInternal) SetLastLogin(t *time.Time) {
	u.LastLoginAt = t
}

func (u *UserInternal) GetSystemRoleID() uint {
	return u.SystemRoleID
}

func (u *UserInternal) GetUsername() string {
	return u.Username
}

func (u *UserInternal) GetUserKind() string {
	return "user"
}

func (u *UserInternal) GetEmail() string {
	return u.Email
}

func (u *UserInternal) GetSource() string {
	return "user"
}

// UserInternal 需要被缓存，实现binary.Marshaler相关接口

func (i *UserInternal) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}

func (i *UserInternal) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, i)
}
