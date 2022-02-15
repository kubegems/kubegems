package orm

import (
	"time"

	"gorm.io/datatypes"
)

// Environment 审计表
// +gen type:object pkcolume:id pkfield:ID preloads:Cluster
type Environment struct {
	ID              uint   `gorm:"primarykey"`
	EnvironmentName string `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_env;index:environment_uniq,unique"`
	Namespace       string `gorm:"type:varchar(50)"`
	Remark          string
	MetaType        string
	DeletePolicy    string `sql:"DEFAULT:'delNamespace'"`

	Creator       *User
	Cluster       *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Project       *Project `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ResourceQuota datatypes.JSON
	LimitRange    datatypes.JSON
	ProjectID     uint `gorm:"uniqueIndex:uniq_idx_project_env"`
	ClusterID     uint
	CreatorID     uint
	Applications  []*Application `gorm:"many2many:application_environment_rels;"`
	Users         []*User        `gorm:"many2many:environment_user_rels;"`

	// 虚拟空间
	VirtualSpaceID *uint
	VirtualSpace   *VirtualSpace `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:SET NULL;"`
}

// EnvironmentUserRels 环境用户关联关系表
// +gen type:objectrel pkcolume:id pkfield:ID preloads:User,Environment leftfield:User rightfield:Environment
type EnvironmentUserRel struct {
	ID          uint         `gorm:"primarykey"`
	User        *User        `json:",omitempty"`
	Environment *Environment `json:"omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 用户ID
	UserID uint `gorm:"uniqueIndex:uniq_idx_env_user_rel" binding:"required"`
	// EnvironmentID
	EnvironmentID uint `gorm:"uniqueIndex:uniq_idx_env_user_rel" binding:"required"`

	// 环境级角色("reader", "operator")
	Role string `binding:"required,eq=reader|eq=operator"`
}

// EnvironmentResource 环境资源统计
// +gen type:object pkcolume:id pkfield:ID
type EnvironmentResource struct {
	ID        uint       `gorm:"primarykey"`
	CreatedAt *time.Time `sql:"DEFAULT:'current_timestamp'"`

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
