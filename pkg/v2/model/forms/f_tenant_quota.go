package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:TenantResourceQuota
type TenantResourceQuotaCommon struct {
	BaseForm
	ID        uint           `json:"id,omitempty"`
	Content   datatypes.JSON `json:"content,omitempty"`
	TenantID  uint           `json:"tenantID,omitempty"`
	ClusterID uint           `json:"clusterID,omitempty"`
	Tenant    *TenantCommon  `json:"tenant,omitempty"`
	Cluster   *ClusterCommon `json:"cluster,omitempty"`
}

// +genform object:TenantResourceQuotaApply
type TenantResourceQuotaApplyCommon struct {
	BaseForm
	ID        uint           `json:"id,omitempty"`
	Content   datatypes.JSON `json:"content,omitempty"`
	Status    string         `json:"status,omitempty"`
	CreateAt  *time.Time     `json:"updatedAt,omitempty"`
	TenantID  uint           `json:"tenantID,omitempty"`
	ClusterID uint           `json:"clusterID,omitempty"`
	Tenant    *TenantCommon  `json:"tenant,omitempty"`
	Cluster   *ClusterCommon `json:"cluster,omitempty"`
	Creator   *UserCommon    `json:"creator,omitempty"`
	CreatorID uint           `json:"creatorID,omitempty"`
}
