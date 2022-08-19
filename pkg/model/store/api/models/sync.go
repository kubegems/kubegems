// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
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
	Status     string     `json:"status"`
	Progress   string     `json:"progress"`
	StartedAt  *time.Time `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}

func SyncStatusFrom(status *SyncServiceSyncStatus) SyncStatus {
	s := SyncStatus{
		Status:   status.State,
		Progress: status.Progress,
	}
	if status.Started != 0 {
		start := time.Unix(status.Started, 0)
		s.StartedAt = &start
	}
	if status.End != 0 {
		end := time.Unix(status.End, 0)
		s.FinishedAt = &end
	}
	return s
}

func (m *ModelsAPI) SyncModel(req *restful.Request, resp *restful.Response) {
	source, name := DecodeSourceModelName(req)
	if msg, err := m.SyncService.SyncOne(req.Request.Context(), source, name); err != nil {
		response.Error(resp, err)
	} else {
		response.OK(resp, msg)
	}
}

func (m *ModelsAPI) SyncStatus(req *restful.Request, resp *restful.Response) {
	source := req.PathParameter("source")
	syncstatus, err := m.SyncService.SyncStatus(req.Request.Context(), source)
	if err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, SyncStatusFrom(syncstatus))
}

func (m *ModelsAPI) StartSync(req *restful.Request, resp *restful.Response) {
	source := req.PathParameter("source")
	if err := m.SyncService.Sync(req.Request.Context(), source, req.Request.URL.Query()); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, nil)
}

func (m *ModelsAPI) StopSync(req *restful.Request, resp *restful.Response) {
	source := req.PathParameter("source")
	if err := m.SyncService.Stop(req.Request.Context(), source, req.Request.URL.Query()); err != nil {
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

func (s *SyncService) Sync(ctx context.Context, source string, query url.Values) error {
	return s.do(ctx, http.MethodPost, fmt.Sprintf("/tasks/start/%s?%s", source, query.Encode()), nil, nil)
}

func (s *SyncService) SyncOne(ctx context.Context, source string, name string) (any, error) {
	msg := &map[string]any{}

	query := url.Values{}
	query.Set("name", name)
	query.Set("source", source)

	if err := s.do(ctx, http.MethodPost, fmt.Sprintf("/sync-one?%s", query.Encode()), nil, msg); err != nil {
		return "", err
	}
	return msg, nil
}

func (s *SyncService) Stop(ctx context.Context, source string, query url.Values) error {
	return s.do(ctx, http.MethodPost, fmt.Sprintf("/tasks/stop/%s?%s", source, query.Encode()), nil, nil)
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
