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
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/v2/models"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
	"kubegems.io/kubegems/pkg/v2/services/handlers/base"
)

var userTags = []string{"users"}

type Handler struct {
	base.BaseHandler
}

func (h *Handler) CreateUser(req *restful.Request, resp *restful.Response) {
	user := &models.UserCreate{}
	if err := handlers.BindData(req, user); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	pass, _ := utils.MakePassword(user.Password)
	newUser := &models.User{
		Email:        user.Email,
		Username:     user.Username,
		Password:     pass,
		SystemRoleID: 2,
	}
	if err := h.DB().WithContext(req.Request.Context()).Create(newUser).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	user.ID = newUser.ID
	handlers.Created(resp, user)
}

func (h *Handler) ListUser(req *restful.Request, resp *restful.Response) {
	userList := []models.UserCommon{}
	if err := h.DB().WithContext(req.Request.Context()).Find(userList).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, userList)
}

func (h *Handler) RetrieveUser(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	user := &models.UserCommon{}
	if err := h.DB().WithContext(ctx).First(user, "username = ?", req.PathParameter("name")).Error; err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
	}
	handlers.OK(resp, user)
}

func (h *Handler) DeleteUser(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	user := models.UserCommon{}
	if err := h.DB().WithContext(ctx).Delete(user, "username = ?", req.PathParameter("name")).Error; err != nil {
		if handlers.IsNotFound(err) {
			handlers.NoContent(resp, nil)
		} else {
			handlers.BadRequest(resp, err)
		}
		return
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) PutUser(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	user := &models.UserCommon{}
	if err := h.DB().WithContext(ctx).First(user, "username = ?", req.PathParameter("name")).Error; err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
	}
	newUser := &models.UserCommon{}
	if err := handlers.BindData(req, newUser); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	user.Email = newUser.Email
	user.Phone = newUser.Phone
	if err := h.DB().WithContext(ctx).Save(user).Error; err != nil {
		handlers.BadRequest(resp, err)
	}
	handlers.OK(resp, user)

}
