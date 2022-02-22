package forms

import "time"

// +genform object:ChartRepo
type ChartRepoCommon struct {
	BaseForm
	ID          uint       `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	URL         string     `json:"url,omitempty"`
	LastSync    *time.Time `json:"lastSync,omitempty"`
	SyncStatus  string     `json:"syncStatus,omitempty"`
	SyncMessage string     `json:"syncMessage,omitempty"`
}
