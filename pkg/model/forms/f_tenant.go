package forms

// +genform object:Tenant
type TenantCommon struct {
	BaseForm
	ID         uint
	TenantName string
}

// +genform object:Tenant
type TenantDetail struct {
	BaseForm
	ID         uint
	TenantName string
	Remark     string
	IsActive   bool
	Users      []*UserCommon
}

// +genform object:TenantUserRel
type TenantUserRelCommon struct {
	BaseForm
	ID       uint
	Tenant   *TenantCommon
	TenantID uint
	User     *UserCommon
	UserID   uint
	Role     string
}
