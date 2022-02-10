package models

import (
	"time"
)

// EnvironmentResource Project资源使用清单
type EnvironmentResource struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`

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
