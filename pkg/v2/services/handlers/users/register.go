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

package userhandler

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/v2/models"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
)

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/v2/users")
	ws.Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(handlers.ListCommonQuery(ws.GET("/").
		To(h.ListUser).
		Doc("list users").
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Returns(http.StatusOK, handlers.MessageOK, UserListResp{})))

	ws.Route(ws.POST("/").
		To(h.CreateUser).
		Doc("create user").
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Reads(models.User{}).
		Returns(http.StatusBadRequest, "validate failed", handlers.Response{}).
		Returns(http.StatusOK, handlers.MessageOK, UserCreateResp{}))

	ws.Route(ws.DELETE("/{name}").
		To(h.DeleteUser).
		Doc("delete user").
		Param(restful.PathParameter("name", "user name")).
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{name}").
		To(h.RetrieveUser).
		Doc("retrieve user").
		Param(restful.PathParameter("name", "user name")).
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Returns(http.StatusOK, handlers.MessageOK, UserCommonResp{}))

	container.Add(ws)
}
