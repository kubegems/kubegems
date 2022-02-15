package forms

// +genform pkg:orm object:SystemRole
type SystemRoleCommon struct {
	BaseForm
	ID       uint   `json:"id"`
	RoleName string `json:"role_name" binding:"required"`
	RoleCode string `json:"role_code" binding:"required"`
}

// +genform pkg:orm object:SystemRole
type SystemRoleDetail struct {
	BaseForm
	ID       uint          `json:"id"`
	RoleName string        `json:"role_name" binding:"required"`
	RoleCode string        `json:"role_code" binding:"required"`
	Users    []*UserCommon `json:"users"`
}
