package models

import (
	"time"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

type SyncStatus struct {
	Status    string    `json:"status"`
	Progress  string    `json:"progress"`
	StartedAt time.Time `json:"startedAt,omitempty"`
}

func (m *ModelsAPI) SyncStatus(req *restful.Request, resp *restful.Response) {
	status := SyncStatus{
		Status:    "running",
		Progress:  "4/16",
		StartedAt: time.Now(),
	}
	response.OK(resp, status)
}

func (m *ModelsAPI) StartSync(req *restful.Request, resp *restful.Response) {
}

func (m *ModelsAPI) StopSync(req *restful.Request, resp *restful.Response) {
}
