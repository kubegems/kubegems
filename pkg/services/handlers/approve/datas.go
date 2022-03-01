package approvehandler

import (
	"time"

	"kubegems.io/pkg/services/handlers"
)

type Approve struct {
	Title   string      `json:"title,omitempty"`
	Kind    ApplyKind   `json:"kind,omitempty"`
	KindID  uint        `json:"recordID,omitempty"`
	Content interface{} `json:"content,omitempty"`
	Time    time.Time   `json:"time,omitempty"`
	Status  string      `json:"status,omitempty"`
}

type ApproveListResp struct {
	handlers.ListBase
	Data []Approve `json:"list"`
}
