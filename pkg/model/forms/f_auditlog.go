package forms

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// +genform object:AuditLog
type AuditLogCommon struct {
	BaseForm
	ID        uint
	CreatedAt *time.Time
	UpdatedAt *time.Time
	DeletedAt gorm.DeletedAt
	Username  string
	Tenant    string
	Module    string
	Name      string
	Action    string
	Success   bool
	ClientIP  string
	Labels    datatypes.JSON
	RawData   datatypes.JSON
}
