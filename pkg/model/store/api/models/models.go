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
	"encoding/base64"
	"net/url"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

func DecodeSourceModelName(req *restful.Request) (string, string) {
	source := req.PathParameter("source")
	name := req.PathParameter("model")

	// model name may contains '/' so we b64encode model name at frontend
	if decoded, err := base64.StdEncoding.DecodeString(name); err == nil {
		name = string(decoded)
	}

	if decodedname, _ := url.PathUnescape(name); decodedname != "" {
		name = decodedname
	}
	return source, name
}

type ModelResponse struct {
	repository.Model
	Rating   repository.Rating `json:"rating"`
	Versions []string          `json:"versions"` // not used
}

func (m *ModelsAPI) ListModels(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	listOptions := repository.ModelListOptions{
		CommonListOptions: ParseCommonListOptions(req),
		Tags:              request.Query(req.Request, "tags", []string{}),
		Framework:         req.QueryParameter("framework"),
		Source:            req.PathParameter("source"),
		WithRating:        request.Query(req.Request, "withRating", true),
		License:           request.Query(req.Request, "license", ""),
		Task:              request.Query(req.Request, "task", ""),
	}

	list, err := m.ModelRepository.List(ctx, listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	// ignore total count error
	total, _ := m.ModelRepository.Count(ctx, listOptions)
	response.OK(resp, response.Page{
		List:  list,
		Total: total,
		Page:  listOptions.Page,
		Size:  listOptions.Size,
	})
}

func (m *ModelsAPI) GetModel(req *restful.Request, resp *restful.Response) {
	source, name := DecodeSourceModelName(req)
	model, err := m.ModelRepository.Get(req.Request.Context(), source, name, false)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, model)
}

func (m *ModelsAPI) DeleteModel(req *restful.Request, resp *restful.Response) {
	source, name := DecodeSourceModelName(req)
	if err := m.ModelRepository.Delete(req.Request.Context(), source, name); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, nil)
}

func (m *ModelsAPI) ListVersions(req *restful.Request, resp *restful.Response) {
	source, name := DecodeSourceModelName(req)
	versions, err := m.ModelRepository.ListVersions(req.Request.Context(), source, name)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, versions)
}

func (m *ModelsAPI) GetVersion(req *restful.Request, resp *restful.Response) {
	source, name := DecodeSourceModelName(req)
	version := req.PathParameter("version")
	model, err := m.ModelRepository.GetVersion(req.Request.Context(), source, name, version)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, model)
}
