package models

import (
	"time"

	"gorm.io/datatypes"
)

/*
ALTER TABLE applications RENAME application_name TO name
*/

// Application 应用表
type Application struct {
	ID            uint         `gorm:"primarykey"`
	Name          string       `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_applicationname;<-:create"`
	Environment   *Environment `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	EnvironmentID *uint        `gorm:"uniqueIndex:uniq_idx_project_applicationname;"`
	Project       *Project     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ProjectID     uint         `gorm:"uniqueIndex:uniq_idx_project_applicationname"`
	Remark        string
	Kind          string
	Images        datatypes.JSON
	Labels        datatypes.JSON
	Creator       string
	CreatedAt     time.Time `sql:"DEFAULT:'current_timestamp'"`
}
