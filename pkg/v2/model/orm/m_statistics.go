package orm

import "time"

// Workload workload resource stastics（for workload resoure suggestion)
// +gen type:object pkcolume:id pkfield:ID preloads:Containers
type Workload struct {
	ID                uint `gorm:"primarykey"`
	Name              string
	Cluster           string
	Namespace         string
	Type              string
	CPULimitStdvar    float64
	MemoryLimitStdvar float64
	CreatedAt         *time.Time   `sql:"DEFAULT:'current_timestamp'"`
	Containers        []*Container `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

// +gen type:object pkcolume:id pkfield:ID
type Container struct {
	ID               uint `gorm:"primarykey"`
	Name             string
	PodName          string
	CPULimitCore     float64
	MemoryLimitBytes int64
	CPUUsageCore     float64
	CPUPercent       float64
	MemoryUsageBytes float64
	MemoryPercent    float64
	WorkloadID       uint
	Workload         *Workload `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}
