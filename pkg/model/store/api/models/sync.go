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
	"strings"
	"sync"
	"time"

	"github.com/emicklei/go-restful/v3"
	"golang.org/x/exp/slices"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/library/rest/response"
	"kubegems.io/modelx/cmd/modelx/model"
	"kubegems.io/modelx/pkg/client"
	"kubegems.io/modelx/pkg/types"
	"sigs.k8s.io/yaml"
)

type SyncOptions struct {
	Addr string `json:"addr" description:"address of the sync service"`
}

func NewDefaultSyncOptions() *SyncOptions {
	return &SyncOptions{
		Addr: "http://kubegems-models-sync:8080",
	}
}

const (
	SyncStatusRunning SyncStatusPhase = "PROGRESS"
	SyncStatusStopped SyncStatusPhase = "STOP"
	SyncStatusSuccess SyncStatusPhase = "SUCCESS"
	SyncStatusFailed  SyncStatusPhase = "FAILURE"
)

type SyncStatusPhase string

type SyncStatus struct {
	Status     SyncStatusPhase `json:"status"`
	Progress   string          `json:"progress"` // eg. 5/34
	StartedAt  *time.Time      `json:"startedAt"`
	FinishedAt *time.Time      `json:"finishedAt"`
	Message    string          `json:"message"`
}

func SyncStatusFrom(status *SyncServiceSyncStatus) *SyncStatus {
	s := &SyncStatus{
		Status:   SyncStatusPhase(status.State),
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
	sourcename, repo := DecodeSourceModelName(req)
	ctx := req.Request.Context()
	source, err := m.SourcesRepository.Get(ctx, sourcename, repository.GetSourceOptions{
		WithAuth: true,
	})
	if err != nil {
		response.Error(resp, err)
		return
	}
	if msg, err := m.SyncService.SyncOne(ctx, source, repo); err != nil {
		response.Error(resp, err)
	} else {
		response.OK(resp, msg)
	}
}

func (m *ModelsAPI) SyncStatus(req *restful.Request, resp *restful.Response) {
	sourcename := req.PathParameter("source")
	ctx := req.Request.Context()
	source, err := m.SourcesRepository.Get(ctx, sourcename, repository.GetSourceOptions{})
	if err != nil {
		response.Error(resp, err)
		return
	}
	syncstatus, err := m.SyncService.SyncStatus(req.Request.Context(), source)
	if err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, syncstatus)
}

func (m *ModelsAPI) StartSync(req *restful.Request, resp *restful.Response) {
	sourcename := req.PathParameter("source")
	ctx := req.Request.Context()
	source, err := m.SourcesRepository.Get(ctx, sourcename, repository.GetSourceOptions{
		WithAuth: true,
	})
	if err != nil {
		response.Error(resp, err)
		return
	}
	if err := m.SyncService.Sync(ctx, source, req.Request.URL.Query()); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, nil)
}

func (m *ModelsAPI) StopSync(req *restful.Request, resp *restful.Response) {
	sourcename := req.PathParameter("source")
	ctx := req.Request.Context()
	source, err := m.SourcesRepository.Get(ctx, sourcename, repository.GetSourceOptions{})
	if err != nil {
		response.Error(resp, err)
		return
	}
	if err := m.SyncService.Stop(ctx, source, req.Request.URL.Query()); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, nil)
}

func NewSyncService(syncopt *SyncOptions, sources *repository.SourcesRepository, models *repository.ModelsRepository) *SyncService {
	return &SyncService{
		opts:    syncopt,
		sources: sources,
		modelx: &ModelxSync{
			jobs:   map[string]*syncjob{},
			models: models,
		},
	}
}

type SyncService struct {
	sources *repository.SourcesRepository
	opts    *SyncOptions
	modelx  *ModelxSync
}

type SyncServiceSyncStatus struct {
	Started  int64 // unix timestamp
	End      int64 // unix timestamp
	State    string
	Progress string
}

func (s *SyncService) SyncStatus(ctx context.Context, source *repository.SourceWithAddtional) (*SyncStatus, error) {
	if source.Kind == repository.SourceKindModelx {
		return s.modelx.SyncStatus(ctx, source)
	}
	status := &SyncServiceSyncStatus{}
	if err := s.do(ctx, http.MethodGet, fmt.Sprintf("/tasks/status/%s", source.Name), nil, status); err != nil {
		return nil, err
	}
	return SyncStatusFrom(status), nil
}

func (s *SyncService) Sync(ctx context.Context, source *repository.SourceWithAddtional, query url.Values) error {
	if source.Kind == repository.SourceKindModelx {
		return s.modelx.Sync(ctx, source)
	}
	return s.do(ctx, http.MethodPost, fmt.Sprintf("/tasks/start/%s?%s", source.Name, query.Encode()), nil, nil)
}

func (s *SyncService) SyncOne(ctx context.Context, source *repository.SourceWithAddtional, repo string) (any, error) {
	if source.Kind == repository.SourceKindModelx {
		return s.modelx.SyncOne(ctx, source, repo)
	}
	msg := &map[string]any{}

	query := url.Values{}
	query.Set("name", repo)
	query.Set("source", source.Name)

	if err := s.do(ctx, http.MethodPost, fmt.Sprintf("/sync-one?%s", query.Encode()), nil, msg); err != nil {
		return "", err
	}
	return msg, nil
}

func (s *SyncService) Stop(ctx context.Context, source *repository.SourceWithAddtional, query url.Values) error {
	if source.Kind == repository.SourceKindModelx {
		return s.modelx.Stop(ctx, source)
	}
	return s.do(ctx, http.MethodPost, fmt.Sprintf("/tasks/stop/%s?%s", source.Name, query.Encode()), nil, nil)
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

type ModelxSync struct {
	jobs   map[string]*syncjob // sourcename + reponame -> job
	mu     sync.Mutex
	models *repository.ModelsRepository
}

func (m *ModelxSync) Sync(ctx context.Context, source *repository.SourceWithAddtional) error {
	go m.getjob(source.Name).syncAll(source.Source)
	return nil
}

func (m *ModelxSync) SyncOne(ctx context.Context, source *repository.SourceWithAddtional, repo string) (any, error) {
	return "ok", syncone(ctx, source.Source, repo, m.models)
}

func (m *ModelxSync) SyncStatus(ctx context.Context, source *repository.SourceWithAddtional) (*SyncStatus, error) {
	return m.getjob(source.Name).status(), nil
}

func (m *ModelxSync) Stop(ctx context.Context, source *repository.SourceWithAddtional) error {
	return m.getjob(source.Name).stop()
}

func (m *ModelxSync) getjob(repo string) *syncjob {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[repo]
	if ok {
		return job
	}
	job = &syncjob{
		stopch: make(chan struct{}),
		target: m.models,
	}
	m.jobs[repo] = job
	return job
}

type syncjob struct {
	stopch    chan struct{} // stop signal
	target    *repository.ModelsRepository
	startAt   time.Time
	finishAt  time.Time
	total     int
	processed int
	done      bool
	errors    map[string]error
}

func (j *syncjob) status() *SyncStatus {
	return &SyncStatus{
		Status: func() SyncStatusPhase {
			if j.startAt.IsZero() {
				return SyncStatusSuccess // not started
			}
			if j.done && j.processed != j.total {
				return SyncStatusStopped
			}
			if len(j.errors) > 0 {
				return SyncStatusFailed
			}
			if !j.done {
				return SyncStatusRunning
			}
			return SyncStatusSuccess
		}(),
		Progress:   fmt.Sprintf("%d/%d", j.processed, j.total),
		StartedAt:  &j.startAt,
		FinishedAt: &j.finishAt,
		Message: func() string {
			ret := ""
			for k, v := range j.errors {
				ret += fmt.Sprintf("%s: %v\n", k, v)
			}
			return ret
		}(),
	}
}

func (j *syncjob) stop() error {
	select {
	case <-j.stopch:
		return nil
	default:
		return nil
	}
}

func (j *syncjob) syncAll(source repository.Source) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-j.stopch
		cancel()
	}()

	j.done = false
	j.startAt = time.Now()
	j.errors = map[string]error{}
	defer func() {
		j.finishAt = time.Now()
		j.done = true
	}()

	cli := client.NewRegistryClient(source.Address, source.Auth.Token)
	globalindex, err := cli.GetGlobalIndex(ctx, "")
	if err != nil {
		j.errors[""] = err
		return
	}
	j.total = len(globalindex.Manifests)
	j.processed = 0
	for _, repo := range globalindex.Manifests {
		log.Info("syncing model", "source", source.Name, "repo", repo.Name)
		if err := syncone(ctx, source, repo.Name, j.target); err != nil {
			log.Error(err, "sync model failed", "source", source.Name, "repo", repo.Name)
			j.errors[repo.Name] = err
		}
		j.processed++
	}
}

func syncone(ctx context.Context, source repository.Source, repo string, target *repository.ModelsRepository) error {
	cli := client.NewRegistryClient(source.Address, source.Auth.Token)
	index, err := cli.GetIndex(ctx, repo, "")
	if err != nil {
		return err
	}
	modelversions := []repository.ModelVersion{}
	var mainConfig *model.ModelConfig
	lastmod := time.Time{}
	slices.SortFunc(index.Manifests, func(i, j types.Descriptor) int {
		return j.Modified.Compare(i.Modified)
	})
	for _, version := range index.Manifests {
		manifest, err := cli.GetManifest(ctx, repo, version.Name)
		if err != nil {
			return err
		}
		if manifest.Config.Modified.After(lastmod) {
			lastmod = manifest.Config.Modified
		}
		if mainConfig == nil {
			content := bytes.NewBuffer(nil)
			if err := cli.GetBlobContent(ctx, repo, manifest.Config.Digest, content); err != nil {
				continue
			}
			thisconfig := &model.ModelConfig{}
			if err := yaml.Unmarshal(content.Bytes(), thisconfig); err != nil {
				continue
			}
			mainConfig = thisconfig
		}

		var files []repository.ModelFile
		readmecontent := bytes.NewBuffer(nil)
		for _, blob := range manifest.Blobs {
			if strings.ToLower(blob.Name) == "readme.md" {
				_ = cli.GetBlobContent(ctx, repo, blob.Digest, readmecontent)
			}
			files = append(files, repository.ModelFile{Filename: blob.Name, Size: blob.Size, ModTime: blob.Modified})
		}
		modelversions = append(modelversions, repository.ModelVersion{
			Name:         version.Name,
			Files:        files,
			Intro:        readmecontent.String(),
			CreationTime: manifest.Config.Modified,
			UpdationTime: manifest.Config.Modified,
		})
	}
	model := &repository.Model{
		Source:       source.Name,
		Name:         repo,
		Versions:     modelversions,
		LastModified: &lastmod,
	}
	if mainConfig != nil {
		model.Framework = mainConfig.FrameWork
		model.Task = mainConfig.Task
		model.Tags = mainConfig.Tags
		model.Author = strings.Join(mainConfig.Mantainers, ",")
	}
	return target.CreateOrUpdateFromSync(ctx, model)
}
