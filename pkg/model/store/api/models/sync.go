package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

type SyncOptions struct {
	Addr string `json:"addr" description:"address of the sync service"`
}

func NewDefaultSyncOptions() *SyncOptions {
	return &SyncOptions{
		Addr: "http://kubegems-models-sync:8080",
	}
}

type SyncStatus struct {
	Status    string    `json:"status"`
	Progress  string    `json:"progress"`
	StartedAt time.Time `json:"startedAt,omitempty"`
}

func (m *ModelsAPI) SyncStatus(req *restful.Request, resp *restful.Response) {
	source := req.PathParameter("source")
	syncstatus, err := m.SyncService.SyncStatus(req.Request.Context(), source)
	if err != nil {
		response.Error(resp, err)
		return
	}
	status := SyncStatus{
		Status:    syncstatus.State,
		Progress:  syncstatus.Progress,
		StartedAt: time.Unix(syncstatus.Started, 0),
	}
	response.OK(resp, status)
}

func (m *ModelsAPI) StartSync(req *restful.Request, resp *restful.Response) {
	source := req.PathParameter("source")
	if err := m.SyncService.StartStop(req.Request.Context(), source, true); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, nil)
}

func (m *ModelsAPI) StopSync(req *restful.Request, resp *restful.Response) {
	source := req.PathParameter("source")
	if err := m.SyncService.StartStop(req.Request.Context(), source, false); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, nil)
}

func NewSyncService(syncopt *SyncOptions) *SyncService {
	return &SyncService{opts: syncopt}
}

type SyncService struct {
	opts *SyncOptions
}

type SyncServiceSyncStatus struct {
	Started  int64 // unix timestamp
	End      int64 // unix timestamp
	State    string
	Progress string
}

func (s *SyncService) SyncStatus(ctx context.Context, source string) (*SyncServiceSyncStatus, error) {
	status := &SyncServiceSyncStatus{}
	if err := s.do(ctx, http.MethodGet, fmt.Sprintf("/tasks/status/%s", source), nil, status); err != nil {
		return nil, err
	}
	return status, nil
}

func (s *SyncService) StartStop(ctx context.Context, source string, start bool) error {
	if start {
		return s.do(ctx, http.MethodPost, fmt.Sprintf("/tasks/start/%s", source), nil, nil)
	} else {
		return s.do(ctx, http.MethodPost, fmt.Sprintf("/tasks/stop/%s", source), nil, nil)
	}
}

func (s *SyncService) do(ctx context.Context, method string, p string, body interface{}, into interface{}) error {
	var bodyreader io.Reader

	switch val := body.(type) {
	case nil:
		bodyreader = nil
	case string:
		bodyreader = bytes.NewBufferString(val)
	case []byte:
		bodyreader = bytes.NewBuffer(val)
	default:
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyreader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, s.opts.Addr+p, bodyreader)
	if err != nil {
		return fmt.Errorf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respbytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status code: %v,body: %s", resp.StatusCode, string(respbytes))
	}
	if into != nil {
		respwrapper := &response.Response{Data: into}
		return json.NewDecoder(resp.Body).Decode(respwrapper)
	}
	return nil
}

func (m *ModelsAPI) registerSourceSyncRoute() *route.Group {
	return route.
		NewGroup("/sync").Tag("sync models").
		AddRoutes(
			route.GET("").To(m.SyncStatus).Doc("sync status").Response(SyncStatus{}),
			route.POST("").To(m.StartSync).Doc("start sync"),
			route.DELETE("").To(m.StopSync).Doc("stop sync"),
		)
}
