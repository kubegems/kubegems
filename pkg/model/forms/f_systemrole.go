package forms

// +genform pkg:orm object:SystemRole
type SystemRoleCommon struct {
	BaseForm
	ID   uint   `json:"id"`
	Name string `json:"roleName" binding:"required"`
	Code string `json:"roleCode" binding:"required"`
}

// +genform pkg:orm object:SystemRole
type SystemRoleDetail struct {
	BaseForm
	ID    uint          `json:"id"`
	Name  string        `json:"roleName" binding:"required"`
	Code  string        `json:"roleCode" binding:"required"`
	Users []*UserCommon `json:"users"`
}
