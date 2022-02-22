package forms

import "time"

// +genform object:VirtualSpace
type VirtualSpaceCommon struct {
	BaseForm
	Name string `json:"name,omitempty"`
	ID   uint   `json:"id,omitempty"`
}

// +genform object:VirtualSpaceUserRel
type VirtualSpaceUserRelCommon struct {
	BaseForm
	ID             uint
	VirtualSpaceID uint
	VirtualSpace   *VirtualSpaceCommon
	UserID         uint
	User           *UserCommon
	Role           string
}

// +genform object:VirtualDomain
type VirtualDomainCommon struct {
	BaseForm
	ID        uint       `json:"id,omitempty"`
	Name      string     `json:"name,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	IsActive  bool       `json:"isActive,omitempty"`
	CreatedBy string     `json:"createdBy,omitempty"`
}
