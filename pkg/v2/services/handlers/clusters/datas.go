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

package clusterhandler

import (
	"kubegems.io/kubegems/pkg/v2/models"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
)

type ClusterInfoResp struct {
	handlers.RespBase
	Data models.ClusterSimple `json:"data"`
}

type ClusterListResp struct {
	handlers.ListBase
	Data []models.ClusterSimple `json:"list"`
}

type LogQueryHistoryListResp struct {
	handlers.ListBase
	Data []models.LogQueryHistory `json:"list"`
}

type LogQuerySnapshotListResp struct {
	handlers.ListBase
	Data []models.LogQuerySnapshot `json:"list"`
}

type ClusterResp struct {
	handlers.RespBase
	Data models.Cluster `json:"data"`
}

type EnvironmentListResp struct {
	handlers.ListBase
	Data []models.Environment `json:"list"`
}

type ClusterQuotaResp struct {
	handlers.RespBase
	Data ClusterQuota `json:"data"`
}

type ClusterStatusMapResp struct {
	handlers.RespBase
	Data ClusterStatusMap `json:"data"`
}
