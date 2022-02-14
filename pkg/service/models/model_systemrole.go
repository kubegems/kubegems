package models

const (
	SystemRoleAdmin    = "sysadmin"
	SystemRoleOrdinary = "ordinary"

	ResSystemRole = "systemrole"
)

// SystemRole 系统角色
type SystemRole struct {
	ID uint `gorm:"primarykey"`
	// 角色名字
	RoleName string
	// 系统级角色Code(管理员admin, 普通用户ordinary)
	RoleCode string `gorm:"type:varchar(30)" binding:"required,eq=sysadmin|eq=normal"`
	Users    []*User
}
