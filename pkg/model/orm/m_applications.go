package orm

import (
	"time"

	"gorm.io/datatypes"
)

// +gen type:object pkcolume:id pkfield:ID preloads:Environment
type Application struct {
	ID            uint       `gorm:"primarykey"`
	Name          string     `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_application;<-:create"`
	CreatedAt     *time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt     *time.Time
	Environment   *Environment `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	EnvironmentID *uint        `gorm:"uniqueIndex:uniq_idx_project_applicationname;"`
	Project       *Project     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ProjectID     uint         `gorm:"uniqueIndex:uniq_idx_project_applicationname"`
	Manifest      datatypes.JSON
	Remark        string
	Kind          string
	Enabled       bool
	Images        datatypes.JSON
	Labels        datatypes.JSON
	Creator       string
}
