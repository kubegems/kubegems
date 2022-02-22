package orm

import "time"

// +gen type:object pkcolume:id pkfield:ID preloads:Cluster,Creator
type LogQueryHistory struct {
	ID         uint     `gorm:"primarykey"`
	Cluster    *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ClusterID  uint
	LabelJSON  string     `gorm:"type:varchar(1024)"`
	FilterJSON string     `gorm:"type:varchar(1024)"`
	LogQL      string     `gorm:"type:varchar(1024)"`
	CreateAt   *time.Time `sql:"DEFAULT:'current_timestamp'"`
	Creator    *User
	CreatorID  uint
}

// +gen type:object pkcolume:id pkfield:ID preloads:Cluster,Creator
type LogQuerySnapshot struct {
	ID            uint     `gorm:"primarykey"`
	Name          string   `gorm:"type:varchar(128)"`
	Cluster       *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ClusterID     uint
	SourceFile    string `gorm:"type:varchar(128)"`
	SnapshotCount int    // file line count
	DownloadURL   string `gorm:"type:varchar(512)"`
	StartTime     *time.Time
	EndTime       *time.Time
	CreateAt      *time.Time `sql:"DEFAULT:'current_timestamp'"`
	Creator       *User
	CreatorID     uint
}
