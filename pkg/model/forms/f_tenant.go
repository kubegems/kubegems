package forms

// +genform object:Tenant
type TenantCommon struct {
	BaseForm
	ID   uint   `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// +genform object:Tenant
type TenantDetail struct {
	BaseForm
	ID       uint          `json:"id,omitempty"`
	Name     string        `json:"name,omitempty"`
	Remark   string        `json:"remark,omitempty"`
	IsActive bool          `json:"isActive,omitempty"`
	Users    []*UserCommon `json:"users,omitempty"`
}

// +genform object:TenantUserRel
type TenantUserRelCommon struct {
	BaseForm
	ID       uint          `json:"id,omitempty"`
	Tenant   *TenantCommon `json:"tenant,omitempty"`
	TenantID uint          `json:"tenantID,omitempty"`
	User     *UserCommon   `json:"user,omitempty"`
	UserID   uint          `json:"userID,omitempty"`
	Role     string        `json:"role,omitempty"`
}

type TenantUserCreateModifyForm struct {
	BaseForm
	Tenant string `json:"tenant" validate:"required"`
	User   string `json:"user" validate:"required"`
	Role   string `json:"role" validate:"required"`
}
