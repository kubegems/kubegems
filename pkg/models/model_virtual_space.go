package models

import "time"

// {
//     "meta": {
//         "name": "hello",
//         "creationTimestamp": "2021-10-13T09:38:18Z",
//         "labels": {
//             "gems.cloudminds.com/creator": "admin"
//         }
//     },
//     "spec": {
//         "Environments": [
//             {
//                 "tenant": "tenant1",
//                 "project": "project2",
//                 "name": "ssss"
//             }
//         ],
//         "DomainRef": null
//     },
//     "status": {
//         "environments": [
//             {
//                 "phase": "OK",
//                 "environment": {
//                     "tenant": "tenant1",
//                     "project": "project2",
//                     "name": "ssss"
//                 }
//             }
//         ]
//     }
// }

const (
	ResVirtualSpace        = "virtualSpace"
	VirtualSpaceRoleAdmin  = "admin"
	VirtualSpaceRoleNormal = "normal"
)

type VirtualSpace struct {
	ID               uint   `gorm:"primarykey"`
	VirtualSpaceName string `gorm:"type:varchar(50);uniqueIndex"`

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
