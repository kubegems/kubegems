package models

import (
	"time"
)

/*
ALTER TABLE registries RENAME COLUMN registry_name TO name
ALTER TABLE registries RENAME COLUMN registry_address TO address
*/

// Registry image registry
type Registry struct {
	ID         uint   `gorm:"primarykey"`
	Name       string `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_registry;"`
	Address    string `gorm:"type:varchar(512)"`
	Username   string `gorm:"type:varchar(50)"`
	Password   string `gorm:"type:varchar(512)"`
	Creator    *User
	CreatorID  uint
	UpdateTime time.Time
	Project    *Project `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ProjectID  uint     `grom:"uniqueIndex:uniq_idx_project_registry;"`
	IsDefault  bool
}
