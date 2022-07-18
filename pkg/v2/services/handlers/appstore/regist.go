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
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/v2/models"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
)

var appStoreTags = []string{"appstore"}

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/v2/appstore")
	ws.Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/repos").
		To(h.ListExternalRepos).
		Doc("list external repos").
		Metadata(restfulspec.KeyOpenAPITags, appStoreTags).
		Returns(http.StatusBadRequest, "list chart repo failed", handlers.Response{}).
		Returns(http.StatusOK, handlers.MessageOK, ChartRepoListResp{}))

	ws.Route(ws.POST("/repos").
		To(h.CreateExternalRepo).
		Doc("create external repos").
		Reads(models.ChartRepo{}).
		Metadata(restfulspec.KeyOpenAPITags, appStoreTags).
		Returns(http.StatusBadRequest, "validate failed", handlers.Response{}).
		Returns(http.StatusOK, handlers.MessageOK, ChartRepoResp{}))

	ws.Route(ws.DELETE("/repos/{repo}").
		To(h.DeleteExternalRepo).
		Doc("delete external repos").
		Param(restful.PathParameter("repo", "reponame")).
		Metadata(restfulspec.KeyOpenAPITags, appStoreTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))

	ws.Route(ws.POST("/repos/{repo}/actions/sync").
		To(h.SyncExternalRepo).
		Doc("sync external repos").
		Param(restful.PathParameter("repo", "reponame")).
		Metadata(restfulspec.KeyOpenAPITags, appStoreTags).
		Returns(http.StatusNotFound, "repo no exist", handlers.Response{}).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.GET("/repos/{repo}/charts").
		To(h.ListApps).
		Doc("list repo charts").
		Param(restful.PathParameter("repo", "reponame").DefaultValue("gems")).
		Metadata(restfulspec.KeyOpenAPITags, appStoreTags).
		Returns(http.StatusBadRequest, "failed to list apps from repo", handlers.Response{}).
		Returns(http.StatusOK, handlers.MessageOK, AppListInfoResp{}))

	ws.Route(ws.GET("/repos/{repo}/charts/{chart}").
		To(h.RetrieveApp).
		Doc("get chart detail info").
		Param(restful.PathParameter("repo", "reponame").DefaultValue("gems")).
		Param(restful.PathParameter("chart", "chart name")).
		Metadata(restfulspec.KeyOpenAPITags, appStoreTags).
		Returns(http.StatusBadRequest, "failed to list apps versions from repo", handlers.Response{}).
		Returns(http.StatusNotFound, "repo not exist", handlers.Response{}).
		Returns(http.StatusOK, handlers.MessageOK, AppListInfoResp{}))

	ws.Route(ws.GET("/repos/{repo}/charts/{chart}/versions/{version}").
		To(h.RetrieveAppFiles).
		Doc("get chart specify version files").
		Param(restful.PathParameter("repo", "reponame").DefaultValue("gems")).
		Param(restful.PathParameter("chart", "chart name")).
		Param(restful.PathParameter("version", "chart version")).
		Metadata(restfulspec.KeyOpenAPITags, appStoreTags).
		Returns(http.StatusBadRequest, "failed to get app files", handlers.Response{}).
		Returns(http.StatusOK, handlers.MessageOK, AppFilesResp{}))

	container.Add(ws)
}
