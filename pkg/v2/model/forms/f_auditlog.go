package forms

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// +genform object:AuditLog
type AuditLogCommon struct {
	BaseForm
	ID        uint           `json:"id,omitempty"`
	CreatedAt *time.Time     `json:"createdAt,omitempty"`
	UpdatedAt *time.Time     `json:"updatedAt,omitempty"`
	DeletedAt gorm.DeletedAt `json:"deletedAt,omitempty"`
	Username  string         `json:"username,omitempty"`
	Tenant    string         `json:"tenant,omitempty"`
	Module    string         `json:"module,omitempty"`
	Name      string         `json:"name,omitempty"`
	Action    string         `json:"action,omitempty"`
	Success   bool           `json:"success,omitempty"`
	ClientIP  string         `json:"clientIP,omitempty"`
	Labels    datatypes.JSON `json:"labels,omitempty"`
	RawData   datatypes.JSON `json:"rawData,omitempty"`
}
