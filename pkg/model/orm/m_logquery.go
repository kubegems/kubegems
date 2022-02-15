package orm

import "time"

// +gen type:object pkcolume:id pkfield:ID preloads:Cluster,Creator
type LogQueryHistory struct {
	ID uint `gorm:"primarykey"`
	// 关联的集群
	Cluster *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 所属集群ID
	ClusterID uint
	// 标签
	LabelJSON string `gorm:"type:varchar(1024)"`
	// 正则标签
	FilterJSON string `gorm:"type:varchar(1024)"`
	// logql
	LogQL string `gorm:"type:varchar(1024)"`
	// 创建时间
	CreateAt *time.Time `sql:"DEFAULT:'current_timestamp'"`
	// 创建者
	Creator   *User
	CreatorID uint
}

// +gen type:object pkcolume:id pkfield:ID preloads:Cluster,Creator
type LogQuerySnapshot struct {
	ID uint `gorm:"primarykey"`
	// 关联的集群
	Cluster *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 所属集群ID
	ClusterID uint
	// 名称
	SnapshotName string `gorm:"type:varchar(128)"`
	SourceFile   string `gorm:"type:varchar(128)"`
	// 行数
	SnapshotCount int
	// 下载地址
	DownloadURL string `gorm:"type:varchar(512)"`
	StartTime   *time.Time
	EndTime     *time.Time
	// 创建时间
	CreateAt *time.Time `sql:"DEFAULT:'current_timestamp'"`
	// 创建者
	Creator   *User
	CreatorID uint
}
