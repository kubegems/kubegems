package orm

import "time"

// Workload workload资源统计表（用于资源建议)
// +gen type:object pkcolume:id pkfield:ID preloads:Containers
type Workload struct {
	ID        uint       `gorm:"primarykey"`
	CreatedAt *time.Time `sql:"DEFAULT:'current_timestamp'"`

	ClusterName       string
	Namespace         string
	Type              string
	Name              string
	CPULimitStdvar    float64
	MemoryLimitStdvar float64

	Containers []*Container `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
}

// Container 辅助数据结构
// +gen type:object pkcolume:id pkfield:ID
type Container struct {
	ID uint `gorm:"primarykey"`

	Name    string
	PodName string

	CPULimitCore     float64
	MemoryLimitBytes int64 // 限制

	CPUUsageCore float64
	CPUPercent   float64 // 使用率

	MemoryUsageBytes float64
	MemoryPercent    float64 // 使用率

	WorkloadID uint
	Workload   *Workload `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}
