package models

import "time"

var ResChartRepo = "chartrepo"

const (
	SyncStatusRunning = "running"
	SyncStatusError   = "error"
	SyncStatusSuccess = "success"
)

/*
ALTER TABLE chart_repos RENAME chart_repo_name TO name
*/

type ChartRepo struct {
	ID          uint   `gorm:"primarykey"`
	Name        string `gorm:"type:varchar(50);uniqueIndex" binding:"required"`
	URL         string `gorm:"type:varchar(255)"`
	LastSync    *time.Time
	SyncStatus  string
	SyncMessage string
}
