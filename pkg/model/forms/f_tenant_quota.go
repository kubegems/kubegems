package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:TenantResourceQuota
type TenantResourceQuotaCommon struct {
	BaseForm
	ID                         uint
	Content                    datatypes.JSON
	TenantID                   uint
	ClusterID                  uint
	Tenant                     *TenantCommon
	Cluster                    *ClusterCommon
	TenantResourceQuotaApply   *TenantResourceQuotaApplyCommon
	TenantResourceQuotaApplyID uint
}

// +genform object:TenantResourceQuotaApply
type TenantResourceQuotaApplyCommon struct {
	BaseForm
	ID        uint
	Content   datatypes.JSON
	Status    string
	Username  string
	UpdatedAt *time.Time
}
