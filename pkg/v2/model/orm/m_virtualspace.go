package orm

import "time"

// +gen type:object pkcolume:id pkfield:ID
type VirtualSpace struct {
	ID   uint
	Name string `gorm:"uniqueIndex"`
}

// +gen type:objectrel pkcolume:id pkfield:ID
type VirtualSpaceUserRel struct {
	ID             uint          `gorm:"primarykey"`
	VirtualSpaceID uint          `gorm:"uniqueIndex:uniq_idx_virtual_space_user_rel"`
	VirtualSpace   *VirtualSpace `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	UserID         uint          `gorm:"uniqueIndex:uniq_idx_virtual_space_user_rel"`
	User           *User         `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Role           string        `gorm:"type:varchar(30)" binding:"required,eq=admin|eq=normal"`
}

// +gen type:object pkcolume:id pkfield:ID
type VirtualDomain struct {
	ID        uint       `gorm:"primarykey"`
	Name      string     `gorm:"type:varchar(50);uniqueIndex"`
	CreatedAt *time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt *time.Time `sql:"DEFAULT:'current_timestamp'"`
	IsActive  bool
	CreatedBy string
}
