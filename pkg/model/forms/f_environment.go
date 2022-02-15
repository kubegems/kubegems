package forms

import (
	"time"

	"gorm.io/datatypes"
)

// +genform object:Environment
type EnvironmentCommon struct {
	BaseForm
	ID              uint
	EnvironmentName string
	Namespace       string
	Remark          string
	MetaType        string
	DeletePolicy    string
}

// +genform object:Environment
type EnvironmentDetail struct {
	BaseForm
	ID              uint
	EnvironmentName string
	Namespace       string
	Remark          string
	MetaType        string
	DeletePolicy    string
	Creator         *UserCommon
	Cluster         *ClusterCommon
	Project         *ProjectCommon
	ResourceQuota   datatypes.JSON
	LimitRange      datatypes.JSON
	ProjectID       uint
	ClusterID       uint
	CreatorID       uint
	Applications    []*ApplicationCommon
	Users           []*UserCommon
	VirtualSpaceID  *uint
	VirtualSpace    *VirtualSpaceCommon
}

// +genform object:EnvironmentUserRel
type EnvironmentUserRelCommon struct {
	BaseForm
	ID            uint
	User          *UserCommon
	Environment   *EnvironmentCommon
	UserID        uint
	EnvironmentID uint
	Role          string
}

// +genform object:EnvironmentResource
type EnvironmentResourceCommon struct {
	BaseForm
	ID        uint
	CreatedAt *time.Time

	ClusterName     string
	TenantName      string
	ProjectName     string
	EnvironmentName string

	MaxCPUUsageCore    float64
	MaxMemoryUsageByte float64
	MinCPUUsageCore    float64
	MinMemoryUsageByte float64
	AvgCPUUsageCore    float64
	AvgMemoryUsageByte float64
	NetworkReceiveByte float64
	NetworkSendByte    float64

	MaxPVCUsageByte float64
	MinPVCUsageByte float64
	AvgPVCUsageByte float64
}
