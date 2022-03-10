package forms

import "time"

// +genform object:LogQueryHistory
type LogQueryHistoryCommon struct {
	BaseForm
	ID         uint           `json:"id,omitempty"`
	Cluster    *ClusterCommon `json:"cluster,omitempty"`
	ClusterID  uint           `json:"clusterID,omitempty"`
	LabelJSON  string         `json:"labelJSON,omitempty"`
	FilterJSON string         `json:"filterJSON,omitempty"`
	LogQL      string         `json:"logQL,omitempty"`
	CreateAt   *time.Time     `json:"createAt,omitempty"`
	Creator    *UserCommon    `json:"creator,omitempty"`
	CreatorID  uint           `json:"creatorID,omitempty"`
}

// +genform object:LogQuerySnapshot
type LogQuerySnapshotCommon struct {
	BaseForm
	ID            uint           `json:"id,omitempty"`
	Cluster       *ClusterCommon `json:"cluster,omitempty"`
	ClusterID     uint           `json:"clusterID,omitempty"`
	Name          string         `json:"name,omitempty"`
	SourceFile    string         `json:"sourceFile,omitempty"`
	SnapshotCount int            `json:"snapshotCount,omitempty"`
	DownloadURL   string         `json:"downloadURL,omitempty"`
	StartTime     *time.Time     `json:"startTime,omitempty"`
	EndTime       *time.Time     `json:"endTime,omitempty"`
	CreateAt      *time.Time     `json:"createAt,omitempty"`
	Creator       *UserCommon    `json:"creator,omitempty"`
	CreatorID     uint           `json:"creatorID,omitempty"`
}
