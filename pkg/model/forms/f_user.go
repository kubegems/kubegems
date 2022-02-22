package forms

import (
	"encoding/json"
	"time"
)

// +genform object:User
type UserCommon struct {
	BaseForm
	ID    uint   `json:"id"`
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
	Role  string `json:"role,omitempty"`
}

// +genform object:User
type UserDetail struct {
	BaseForm
	ID           uint              `json:"id,omitempty"`
	Name         string            `json:"name,omitempty"`
	Email        string            `json:"email,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	Source       string            `json:"source,omitempty"`
	IsActive     *bool             `json:"isActive,omitempty"`
	CreatedAt    *time.Time        `json:"createdAt,omitempty"`
	LastLoginAt  *time.Time        `json:"lastLoginAt,omitempty"`
	SystemRole   *SystemRoleCommon `json:"systemRole,omitempty"`
	SystemRoleID uint              `json:"systemRoleID,omitempty"`

	Role string `json:"role,omitempty"`
}

// +genform object:User
type UserSetting struct {
	BaseForm
	ID           uint              `json:"id,omitempty"`
	Name         string            `json:"name,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	Email        string            `json:"email,omitempty"`
	Password     string            `json:"password,omitempty"`
	IsActive     *bool             `json:"isActive,omitempty"`
	SystemRole   *SystemRoleCommon `json:"systemRole,omitempty"`
	SystemRoleID uint              `json:"systemRoleID,omitempty"`
}

// +genform object:User
type UserInternal struct {
	BaseForm
	ID           uint              `json:"id,omitempty"`
	Name         string            `json:"name,omitempty"`
	Password     string            `json:"password,omitempty"`
	Email        string            `json:"email,omitempty"`
	Role         string            `json:"role,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	Source       string            `json:"source,omitempty"`
	IsActive     *bool             `json:"isActive,omitempty"`
	CreatedAt    *time.Time        `json:"createdAt,omitempty"`
	LastLoginAt  *time.Time        `json:"lastLoginAt,omitempty"`
	SystemRole   *SystemRoleCommon `json:"systemRole,omitempty"`
	SystemRoleID uint              `json:"systemRoleID,omitempty"`
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
	return u.Name
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
