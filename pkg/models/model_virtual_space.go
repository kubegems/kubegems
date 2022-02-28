package models

import "time"

/*
ALTER TABLE virtual_spaces RENAME COLUMN virtual_space_name TO name;
*/

const (
	ResVirtualSpace        = "virtualSpace"
	VirtualSpaceRoleAdmin  = "admin"
	VirtualSpaceRoleNormal = "normal"
)

type VirtualSpace struct {
	ID   uint   `gorm:"primarykey"`
	Name string `gorm:"type:varchar(50);uniqueIndex"`

	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`

	IsActive  bool
	CreatedBy string

	Users        []*User `gorm:"many2many:virtual_space_user_rels;"`
	Environments []*Environment
}

type VirtualSpaceUserRels struct {
	ID uint `gorm:"primarykey"`

	VirtualSpaceID uint          `gorm:"uniqueIndex:uniq_idx_virtual_space_user_rel"`
	VirtualSpace   *VirtualSpace `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`

	UserID uint  `gorm:"uniqueIndex:uniq_idx_virtual_space_user_rel"`
	User   *User `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`

	// 虚拟空间角色(管理员admin, 普通用户normal)
	Role string `gorm:"type:varchar(30)" binding:"required,eq=admin|eq=normal"`
}
