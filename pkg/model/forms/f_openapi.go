package forms

import (
	"encoding/json"
	"time"
)

// +genform object:OpenAPP
type OpenAPPCommon struct {
	BaseForm
	Name           string
	ID             uint
	AppID          string
	PermScopes     string
	TenantScope    string
	RequestLimiter int
}

// +genform object:OpenAPP
type OpenAPPDetail struct {
	BaseForm
	Name           string
	ID             uint
	AppID          string
	AppSecret      string
	PermScopes     string
	TenantScope    string
	RequestLimiter int
}

func (u *OpenAPPDetail) GetID() uint {
	return u.ID
}

func (u *OpenAPPDetail) SetLastLogin(t *time.Time) {
}

func (u *OpenAPPDetail) GetSystemRoleID() uint {
	return 0
}

func (u *OpenAPPDetail) GetUsername() string {
	return u.Name
}

func (u *OpenAPPDetail) GetUserKind() string {
	return "app"
}

func (u *OpenAPPDetail) GetEmail() string {
	return ""
}

func (u *OpenAPPDetail) GetSource() string {
	return "app"
}

func (i *OpenAPPDetail) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}

func (i *OpenAPPDetail) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, i)
}
