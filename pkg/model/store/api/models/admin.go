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
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

func (m ModelsAPI) AdminListSources(req *restful.Request, resp *restful.Response) {
	listOptions := repository.ListSourceOptions{
		WithDisabled:    true,
		WithModelCounts: true,
	}
	list, err := m.SourcesRepository.List(req.Request.Context(), listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, list)
}

func (m ModelsAPI) AdminCheckSource(req *restful.Request, resp *restful.Response) {
	source := &repository.Source{}
	if err := req.ReadEntity(source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	if err := checkSource(req.Request.Context(), source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, source)
}

func checkSource(ctx context.Context, source *repository.Source) error {
	if source.Kind == repository.SourceKindModelx {
		// check modelx source
		return checkModelxConnection(ctx, source.Address, source.Auth.Token)
	}
	return nil
}

func checkModelxConnection(ctx context.Context, addr string, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		// nolint: gomnd
		bosystr, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("unexpected status code %d,body: %s", resp.StatusCode, string(bosystr))
	}
	return nil
}

func (m ModelsAPI) AdminCreateSource(req *restful.Request, resp *restful.Response) {
	source := &repository.Source{}
	if err := req.ReadEntity(source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	// check
	if err := checkSource(req.Request.Context(), source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	if err := m.SourcesRepository.Create(req.Request.Context(), source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, source)
}

func (m ModelsAPI) AdminGetSource(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("source")
	getsourceopt := repository.GetSourceOptions{
		WithDisabled: true,
		WithCounts:   true,
		WithAuth:     true,
	}
	if source, err := m.SourcesRepository.Get(req.Request.Context(), name, getsourceopt); err != nil {
		response.BadRequest(resp, err.Error())
	} else {
		response.OK(resp, source)
	}
}

func (m ModelsAPI) AdminDeleteSource(req *restful.Request, resp *restful.Response) {
	source := &repository.Source{
		Name: req.PathParameter("source"),
	}
	if err := m.SourcesRepository.Delete(req.Request.Context(), source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, source)
}

func (m ModelsAPI) AdminUpdateSource(req *restful.Request, resp *restful.Response) {
	source := &repository.Source{}
	if err := req.ReadEntity(source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	source.Name = req.PathParameter("source")
	if err := m.SourcesRepository.Update(req.Request.Context(), source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, source)
}

type SimpleModel struct {
	// nolint: tagliatelle
	ID        string   `json:"id,omitempty" bson:"_id,omitempty"`
	Name      string   `json:"name"`
	Tags      []string `json:"tags"`
	Intro     string   `json:"intro"`
	Recomment int      `json:"recomment"` // number of recomment votes
	Disabled  bool     `json:"disabled"`
}

func (m ModelsAPI) AdminListModel(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	listOptions := repository.ModelListOptions{
		CommonListOptions: ParseCommonListOptions(req),
		Tags:              request.Query(req.Request, "tags", []string{}),
		Framework:         req.QueryParameter("framework"),
		Source:            req.PathParameter("source"),
		License:           request.Query(req.Request, "license", ""),
		Task:              request.Query(req.Request, "task", ""),
		WithDisabled:      true,
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

func (m ModelsAPI) AdminUpdateModel(req *restful.Request, resp *restful.Response) {
	model := &repository.Model{}
	if err := req.ReadEntity(model); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	model.Source, model.Name = DecodeSourceModelName(req)
	if err := m.ModelRepository.Update(req.Request.Context(), model); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, model)
}

func (m *ModelsAPI) AdminListSelector(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	listOptions := repository.ModelListOptions{
		CommonListOptions: ParseCommonListOptions(req),
		Tags:              request.Query(req.Request, "tags", []string{}),
		Framework:         req.QueryParameter("framework"),
		Source:            req.PathParameter("source"),
		License:           request.Query(req.Request, "license", ""),
		Task:              request.Query(req.Request, "task", ""),
		WithDisabled:      true,
	}
	selector, err := m.ModelRepository.ListSelectors(ctx, listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, selector)
}

// nolint: funlen
func (m *ModelsAPI) registerAdminRoute() *route.Group {
	return route.
		NewGroup("/admin").Tag("admin").
		AddSubGroup(
			// source
			route.NewGroup("/sources").AddRoutes(
				route.POST("check").To(m.AdminCheckSource).Doc("check source").
					Parameters(
						route.BodyParameter("source", repository.Source{}),
					),
				route.GET("").To(m.AdminListSources).Doc("list sources").
					Response([]repository.SourceWithAddtional{}),
				route.GET("/{source}").To(m.AdminGetSource).Doc("get source").
					Parameters(route.PathParameter("source", "source name")).
					Response(repository.SourceWithAddtional{}),
				route.POST("").To(m.AdminCreateSource).Doc("add source").
					Parameters(
						route.BodyParameter("source", repository.Source{}),
					),
				route.PUT("/{source}").To(m.AdminUpdateSource).Doc("update source").
					Parameters(
						route.PathParameter("source", "source name"),
						route.BodyParameter("source", repository.Source{}),
					).
					Response(repository.Source{}),
				route.DELETE("/{source}").To(m.AdminDeleteSource).Doc("delete source").
					Parameters(route.PathParameter("source", "source name")).
					Response(repository.Source{}),
			),
			// source selector
			route.NewGroup("/sources/{source}/selector").
				Parameters(route.PathParameter("source", "source name")).
				AddRoutes(
					route.GET("").To(m.AdminListSelector).Doc("list selector"),
				),
			// source sync
			route.NewGroup("/sources/{source}/sync").
				Parameters(route.PathParameter("source", "source name")).
				AddRoutes(
					route.GET("").To(m.SyncStatus).Doc("sync status").Response(SyncStatus{}),
					route.POST("").To(m.StartSync).Doc("start sync").Parameters(
						route.QueryParameter("model", "model name to sync"),
					),
					route.DELETE("").To(m.StopSync).Doc("stop sync"),
				),
			// source model
			route.
				NewGroup("/sources/{source}/models").
				Parameters(route.PathParameter("source", "source name")).
				AddRoutes(
					route.GET("").To(m.AdminListModel).Doc("list models").
						Parameters(
							route.QueryParameter("tags", "tags"),
							route.QueryParameter("framework", "framework"),
							route.QueryParameter("license", "license"),
							route.QueryParameter("task", "task"),
							route.QueryParameter("search", "search keyword"),
						).
						Paged().Response([]SimpleModel{}),
					route.PUT("/{model}").To(m.AdminUpdateModel).Doc("update model").Parameters(
						route.PathParameter("model", "model name"),
						route.BodyParameter("body", SimpleModel{}),
					),
				),
		)
}
