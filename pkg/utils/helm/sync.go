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

package helm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"helm.sh/helm/v3/pkg/repo"
	helm_repo "helm.sh/helm/v3/pkg/repo"
	"kubegems.io/kubegems/pkg/log"
)

// MaxSyncVerionCount 最大同步版本，仅同步某个chart的最近的 n 个版本
const MaxSyncVerionCount = 3

type ProcessEvent struct {
	Chart   *repo.ChartVersion
	Error   error
	Message string
}

var syncState sync.Map

var ErrSynchronizing = errors.New("synchronizing")

func SyncChartsToChartmuseum(ctx context.Context, remote RepositoryConfig, localChartMuseum RepositoryConfig) error {
	onProcess := func(e ProcessEvent) {
		if e.Error != nil {
			log.Errorf("sync chart %s:%s, error: %s", e.Chart.Name, e.Chart.Version, e.Error.Error())
		} else {
			log.Infof("sync chart %s:%s,message: %s", e.Chart.Name, e.Chart.Version, e.Message)
		}
	}
	return SyncChartsToChartmuseumWithProcess(ctx, remote, localChartMuseum, onProcess)
}

func SyncChartsToChartmuseumWithProcess(ctx context.Context, remote RepositoryConfig, localChartMuseum RepositoryConfig, onProcess func(ProcessEvent)) error {
	if onProcess == nil {
		onProcess = func(ProcessEvent) {}
	}
	remoteRepository, err := NewLegencyRepository(&remote)
	if err != nil {
		return err
	}
	if remote.Name == "" {
		return errors.New("empty remote repository name")
	}

	// state marchine
	if _, ok := syncState.Load(remote.Name); ok {
		return fmt.Errorf("repo %s %w", remote.Name, ErrSynchronizing)
	}
	syncState.Store(remote.Name, struct{}{})
	defer syncState.Delete(remote.Name)

	index, err := remoteRepository.GetIndex(ctx)
	if err != nil {
		return fmt.Errorf("download index: %w", err)
	}
	chartmuseum, err := NewChartMuseumClient(&localChartMuseum)
	if err != nil {
		return err
	}
	if err := chartmuseum.Health(ctx); err != nil {
		return fmt.Errorf("chartmuseum health check failed: %w", err)
	}

	errors := []error{}
	// 暂时使用 1 并发
	concurrent := NewGoconcurrent(1)

	// get local repo index
	localIndex, err := chartmuseum.ListAllChartVersions(ctx, remote.Name)
	if err != nil {
		log.Warnf("chartmuseum repo %s inedx error: %s", err.Error())
	}

	findLocalVersion := func(name, version string) *helm_repo.ChartVersion {
		if cvs, ok := localIndex[name]; ok {
			for _, v := range cvs {
				if v.Version == version {
					return v
				}
			}
		}
		return nil
	}

	for i := range index.Entries {
		versions := index.Entries[i]
		for j := range versions {
			// j start from 0 ，using >=
			if j >= MaxSyncVerionCount {
				break
			}
			version := versions[j]
			concurrent.Go(func() {
				onProcess(ProcessEvent{Chart: version, Message: "checking"})
				if len(version.URLs) == 0 {
					err := fmt.Errorf("chart version %s no urls found", version.Name)
					errors = append(errors, err)
					onProcess(ProcessEvent{Chart: version, Error: err})
					return
				}

				// 检查Digest，若相同则跳过
				if existVersion := findLocalVersion(version.Name, version.Version); existVersion != nil && existVersion.Digest == version.Digest {
					onProcess(ProcessEvent{Chart: version, Message: "already exist and uptodate"})
					return
				}
				onProcess(ProcessEvent{Chart: version, Message: "downloading"})
				readercloser, err := remoteRepository.GetFile(ctx, version.URLs[0])
				if err != nil {
					errors = append(errors, err)
					onProcess(ProcessEvent{Chart: version, Error: err})
					return
				}
				onProcess(ProcessEvent{Chart: version, Message: "pushing"})
				if err := chartmuseum.UploadChart(ctx, remote.Name, readercloser); err != nil {
					errors = append(errors, err)
					onProcess(ProcessEvent{Chart: version, Error: err})
					return
				}
				readercloser.Close()
				onProcess(ProcessEvent{Chart: version, Message: "synced"})
			})
		}
	}
	if len(errors) > 0 {
		return Errors(errors)
	}
	return nil
}

type Errors []error

func (errs Errors) Error() string {
	builder := strings.Builder{}
	for _, err := range errs {
		builder.WriteString(err.Error())
		builder.WriteString(";")
	}
	return builder.String()
}

func NewGoconcurrent(concurrent int) *Goconcurrent {
	return &Goconcurrent{state: make(chan struct{}, concurrent)}
}

type Goconcurrent struct {
	state chan struct{}
}

func (g *Goconcurrent) Go(f func()) {
	g.state <- struct{}{}
	f()
	<-g.state
}
