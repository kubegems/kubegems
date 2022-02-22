package orm

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// +gen type:object kind:auditlog pkcolume:id pkfield:ID
type AuditLog struct {
	ID        uint       `gorm:"primarykey"`
	Name      string     `gorm:"type:varchar(512)"`
	CreatedAt *time.Time `gorm:"index"`
	UpdatedAt *time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Username  string         `gorm:"type:varchar(50)"`
	Tenant    string         `gorm:"type:varchar(50)"`
	Module    string         `gorm:"type:varchar(512)"`
	Action    string         `gorm:"type:varchar(255)"`
	Success   bool
	ClientIP  string `gorm:"type:varchar(255)"`
	Labels    datatypes.JSON
	RawData   datatypes.JSON
}
