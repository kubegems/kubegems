package models

const (
	SystemRoleAdmin    = "sysadmin"
	SystemRoleOrdinary = "ordinary"

	ResSystemRole = "systemrole"
)

/*
ALTER TABLE system_roles RENAME COLUMN role_name TO name
ALTER TABLE system_roles RENAME COLUMN role_code TO code
*/

type SystemRole struct {
	ID    uint `gorm:"primarykey"`
	Name  string
	Code  string `gorm:"type:varchar(30)" binding:"required,eq=sysadmin|eq=normal"`
	Users []*User
}
