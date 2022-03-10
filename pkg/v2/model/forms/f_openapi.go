package forms

import (
	"encoding/json"
	"time"
)

// +genform object:OpenAPP
type OpenAPPCommon struct {
	BaseForm
	Name           string `json:"name,omitempty"`
	ID             uint   `json:"id,omitempty"`
	AppID          string `json:"appID,omitempty"`
	PermScopes     string `json:"permScopes,omitempty"`
	TenantScope    string `json:"tenantScope,omitempty"`
	RequestLimiter int    `json:"requestLimiter,omitempty"`
}

// +genform object:OpenAPP
type OpenAPPDetail struct {
	BaseForm
	Name           string `json:"name,omitempty"`
	ID             uint   `json:"id,omitempty"`
	AppID          string `json:"appID,omitempty"`
	AppSecret      string `json:"appSecret,omitempty"`
	PermScopes     string `json:"permScopes,omitempty"`
	TenantScope    string `json:"tenantScope,omitempty"`
	RequestLimiter int    `json:"requestLimiter,omitempty"`
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
