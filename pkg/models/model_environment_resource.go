package models

import (
	"time"
)

/*
ALTER TABLE environment_resources RENAME COLUMN cluster_name TO cluster;
ALTER TABLE environment_resources RENAME COLUMN tenant_name TO tenant;
ALTER TABLE environment_resources RENAME COLUMN project_name TO project;
ALTER TABLE environment_resources RENAME COLUMN environment_name TO environment;
*/

// EnvironmentResource Project资源使用清单
type EnvironmentResource struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`

	Cluster     string
	Tenant      string
	Project     string
	Environment string

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
