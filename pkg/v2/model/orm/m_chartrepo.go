package orm

import "time"

// +gen type:object kind:chartrepo pkcolume:id pkfield:ID
type ChartRepo struct {
	ID          uint   `gorm:"primarykey"`
	Name        string `gorm:"type:varchar(50);uniqueIndex"`
	URL         string `gorm:"type:varchar(255)"`
	LastSync    *time.Time
	SyncStatus  string
	SyncMessage string
}
