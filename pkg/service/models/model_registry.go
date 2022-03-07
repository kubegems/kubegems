package models

import (
	"time"
)

// Registry 镜像仓库表
type Registry struct {
	ID uint `gorm:"primarykey"`
	// 仓库名称
	RegistryName string `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_registry;"`
	// 仓库地址
	RegistryAddress string `gorm:"type:varchar(512)"`
	// 用户名
	Username string `gorm:"type:varchar(50)"`
	// 密码
	Password string `gorm:"type:varchar(512)"`
	// 创建者
	Creator *User
	// 更新时间
	UpdateTime time.Time
	CreatorID  uint
	Project    *Project `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 项目ID
	ProjectID     uint `grom:"uniqueIndex:uniq_idx_project_registry;"`
	IsDefault     bool
	EnableExtends bool // 是否启用扩展功能，支持harbor等高级仓库
}
