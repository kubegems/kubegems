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
	"errors"

	"github.com/emicklei/go-restful/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

func (m *ModelsAPI) ListSelectors(req *restful.Request, resp *restful.Response) {
	listOptions := repository.ModelListOptions{
		CommonListOptions: ParseCommonListOptions(req),
		Tags:              request.Query(req.Request, "tags", []string{}),
		Framework:         req.QueryParameter("framework"),
		Source:            req.PathParameter("source"),
	}
	selectors, err := m.ModelRepository.ListSelectors(req.Request.Context(), listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, selectors)
}

type SourceWithSyncStatus struct {
	repository.SourceWithAddtional `json:",inline"`
	SyncStatus                     SyncStatus `json:"syncStatus"`
}

func (m *ModelsAPI) GetSource(req *restful.Request, resp *restful.Response) {
	getsourceopt := repository.GetSourceOptions{
		WithCounts: request.Query(req.Request, "count", false),
	}
	name := req.PathParameter("source")
	source, err := m.SourcesRepository.Get(req.Request.Context(), name, getsourceopt)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(resp, err.Error())
		} else {
			response.BadRequest(resp, err.Error())
		}
		return
	}
	// with sync status
	syncstatus, err := m.SyncService.SyncStatus(req.Request.Context(), source)
	if err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, &SourceWithSyncStatus{
		SourceWithAddtional: *source,
		SyncStatus:          *syncstatus,
	})
}

func (m *ModelsAPI) ListSources(req *restful.Request, resp *restful.Response) {
	listOptions := repository.ListSourceOptions{}
	list, err := m.SourcesRepository.List(req.Request.Context(), listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, list)
}
