package orm

import (
	"time"
)

// +gen type:object pkcolume:id pkfield:ID preloads:Project
type Registry struct {
	ID              uint   `gorm:"primarykey"`
	RegistryName    string `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_registry;"`
	RegistryAddress string `gorm:"type:varchar(512)"`
	Username        string `gorm:"type:varchar(50)"`
	Password        string `gorm:"type:varchar(512)"`
	Creator         *User
	UpdateTime      *time.Time
	CreatorID       uint
	Project         *Project `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ProjectID       uint     `grom:"uniqueIndex:uniq_idx_project_registry;"`
	IsDefault       bool
}
