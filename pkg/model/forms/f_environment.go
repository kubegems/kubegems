package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:Environment
type EnvironmentCommon struct {
	BaseForm
	ID           uint   `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Namespace    string `json:"namespace,omitempty"`
	Remark       string `json:"remark,omitempty"`
	MetaType     string `json:"metaType,omitempty"`
	DeletePolicy string `json:"deletePolicy,omitempty"`
}

// +genform object:Environment
type EnvironmentDetail struct {
	BaseForm
	ID             uint                 `json:"id,omitempty"`
	Name           string               `json:"name,omitempty"`
	Namespace      string               `json:"namespace,omitempty"`
	Remark         string               `json:"remark,omitempty"`
	MetaType       string               `json:"metaType,omitempty"`
	DeletePolicy   string               `json:"deletePolicy,omitempty"`
	Creator        *UserCommon          `json:"creator,omitempty"`
	Cluster        *ClusterCommon       `json:"cluster,omitempty"`
	Project        *ProjectCommon       `json:"project,omitempty"`
	ResourceQuota  datatypes.JSON       `json:"resourceQuota,omitempty"`
	LimitRange     datatypes.JSON       `json:"limitRange,omitempty"`
	ProjectID      uint                 `json:"projectID,omitempty"`
	ClusterID      uint                 `json:"clusterID,omitempty"`
	CreatorID      uint                 `json:"creatorID,omitempty"`
	Applications   []*ApplicationCommon `json:"applications,omitempty"`
	Users          []*UserCommon        `json:"users,omitempty"`
	VirtualSpaceID *uint                `json:"virtualSpaceID,omitempty"`
	VirtualSpace   *VirtualSpaceCommon  `json:"virtualSpace,omitempty"`
}

// +genform object:EnvironmentUserRel
type EnvironmentUserRelCommon struct {
	BaseForm
	ID            uint               `json:"id,omitempty"`
	User          *UserCommon        `json:"user,omitempty"`
	Environment   *EnvironmentCommon `json:"environment,omitempty"`
	UserID        uint               `json:"userID,omitempty"`
	EnvironmentID uint               `json:"environmentID,omitempty"`
	Role          string             `json:"role,omitempty"`
}

// +genform object:EnvironmentResource
type EnvironmentResourceCommon struct {
	BaseForm
	ID        uint       `json:"id,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`

	Cluster     string `json:"cluster,omitempty"`
	Tenant      string `json:"tenant,omitempty"`
	Project     string `json:"project,omitempty"`
	Environment string `json:"environment,omitempty"`

	MaxCPUUsageCore    float64 `json:"maxCPUUsageCore,omitempty"`
	MaxMemoryUsageByte float64 `json:"maxMemoryUsageByte,omitempty"`
	MinCPUUsageCore    float64 `json:"minCPUUsageCore,omitempty"`
	MinMemoryUsageByte float64 `json:"minMemoryUsageByte,omitempty"`
	AvgCPUUsageCore    float64 `json:"avgCPUUsageCore,omitempty"`
	AvgMemoryUsageByte float64 `json:"avgMemoryUsageByte,omitempty"`
	NetworkReceiveByte float64 `json:"networkReceiveByte,omitempty"`
	NetworkSendByte    float64 `json:"networkSendByte,omitempty"`

	MaxPVCUsageByte float64 `json:"maxPVCUsageByte,omitempty"`
	MinPVCUsageByte float64 `json:"minPVCUsageByte,omitempty"`
	AvgPVCUsageByte float64 `json:"avgPVCUsageByte,omitempty"`
}
