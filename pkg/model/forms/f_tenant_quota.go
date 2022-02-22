package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:TenantResourceQuota
type TenantResourceQuotaCommon struct {
	BaseForm
	ID                         uint                            `json:"id,omitempty"`
	Content                    datatypes.JSON                  `json:"content,omitempty"`
	TenantID                   uint                            `json:"tenantID,omitempty"`
	ClusterID                  uint                            `json:"clusterID,omitempty"`
	Tenant                     *TenantCommon                   `json:"tenant,omitempty"`
	Cluster                    *ClusterCommon                  `json:"cluster,omitempty"`
	TenantResourceQuotaApply   *TenantResourceQuotaApplyCommon `json:"tenantResourceQuotaApply,omitempty"`
	TenantResourceQuotaApplyID uint                            `json:"tenantResourceQuotaApplyID,omitempty"`
}

// +genform object:TenantResourceQuotaApply
type TenantResourceQuotaApplyCommon struct {
	BaseForm
	ID        uint           `json:"id,omitempty"`
	Content   datatypes.JSON `json:"content,omitempty"`
	Status    string         `json:"status,omitempty"`
	Username  string         `json:"username,omitempty"`
	UpdatedAt *time.Time     `json:"updatedAt,omitempty"`
}
