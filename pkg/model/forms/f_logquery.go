package forms

import "time"

// +genform object:LogQueryHistory
type LogQueryHistoryCommon struct {
	BaseForm
	ID         uint
	Cluster    *ClusterCommon
	ClusterID  uint
	LabelJSON  string
	FilterJSON string
	LogQL      string
	CreateAt   *time.Time
	Creator    *UserCommon
	CreatorID  uint
}

// +genform object:LogQuerySnapshot
type LogQuerySnapshotCommon struct {
	BaseForm
	ID            uint
	Cluster       *ClusterCommon
	ClusterID     uint
	SnapshotName  string
	SourceFile    string
	SnapshotCount int
	DownloadURL   string
	StartTime     *time.Time
	EndTime       *time.Time
	CreateAt      *time.Time
	Creator       *UserCommon
	CreatorID     uint
}
