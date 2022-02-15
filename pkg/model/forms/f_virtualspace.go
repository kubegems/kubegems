package forms

import "time"

// +genform object:VirtualSpace
type VirtualSpaceCommon struct {
	BaseForm
	VirtualSpaceName string
	ID               uint
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
	ID                uint
	VirtualDomainName string
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
	IsActive          bool
	CreatedBy         string
}
