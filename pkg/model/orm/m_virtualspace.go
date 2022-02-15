package orm

import "time"

// VirtualSpace 虚拟空间
// +gen type:object pkcolume:id pkfield:ID preloads:SystemRole
type VirtualSpace struct {
	ID               uint
	VirtualSpaceName string
}

// +gen type:objectrel pkcolume:id pkfield:ID preloads:SystemRole leftfield:VirtualSpace rightfield:User
type VirtualSpaceUserRel struct {
	ID uint `gorm:"primarykey"`

	VirtualSpaceID uint          `gorm:"uniqueIndex:uniq_idx_virtual_space_user_rel"`
	VirtualSpace   *VirtualSpace `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`

	UserID uint  `gorm:"uniqueIndex:uniq_idx_virtual_space_user_rel"`
	User   *User `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`

	// 虚拟空间角色(管理员admin, 普通用户normal)
	Role string `gorm:"type:varchar(30)" binding:"required,eq=admin|eq=normal"`
}

// VirtualDomainName virtualdomain
// +gen type:object pkcolume:id pkfield:ID
type VirtualDomain struct {
	ID                uint   `gorm:"primarykey"`
	VirtualDomainName string `gorm:"type:varchar(50);uniqueIndex"`

	CreatedAt *time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt *time.Time `sql:"DEFAULT:'current_timestamp'"`

	IsActive  bool // 是否激活
	CreatedBy string
}
