package forms

import "time"

// +genform object:ChartRepo
type ChartRepoCommon struct {
	BaseForm
	ID            uint
	ChartRepoName string
	URL           string
	LastSync      *time.Time
	SyncStatus    string
	SyncMessage   string
}
