package orm

// +gen type:object pkcolume:id pkfield:ID preloads:Users
type SystemRole struct {
	ID       uint `gorm:"primary_key"`
	RoleName string
	RoleCode string `gorm:"type:varchar(30)" binding:"required;eq=admin|eq=ordinary"`
	Users    []*User
}
