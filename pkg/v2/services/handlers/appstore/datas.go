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

package appstorehandler

import (
	"time"

	"helm.sh/helm/v3/pkg/chart"
	"kubegems.io/kubegems/pkg/v2/models"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
)

type Chart struct {
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Description string              `json:"description"`
	ApiVersion  string              `json:"apiVersion"`
	AppVersion  string              `json:"appVersion"`
	Created     time.Time           `json:"created"`
	Digest      string              `json:"digest"`
	Urls        []string            `json:"urls"`
	Keywords    []string            `json:"keywords"`
	Maintainers []*chart.Maintainer `json:"maintainers"`
	Tags        string              `json:"tags"`
	RepoURL     string              `json:"repoURL"`
}

type AppFiles struct {
	Files   map[string]string `json:"files" description:"files"`
	App     string            `json:"app" description:"app"`
	Version string            `json:"version" description:"version"`
}

type AppListInfoResp struct {
	handlers.RespBase
	Data []Chart `json:"data"`
}

type AppFilesResp struct {
	handlers.RespBase
	Data AppFiles `json:"data"`
}

type ChartRepoListResp struct {
	handlers.ListBase
	Data []models.ChartRepo `json:"list"`
}

type ChartRepoResp struct {
	handlers.RespBase
	Data []models.ChartRepo `json:"data"`
}
