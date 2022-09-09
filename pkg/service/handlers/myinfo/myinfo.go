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

package myinfohandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils"
)

// Myinfo 获取当前用户的信息
// @Tags        User
// @Summary     获取当前用户的信息
// @Description 获取当前用户的信息
// @Accept      json
// @Produce     json
// @Success     200 {object} handlers.ResponseStruct{Data=[]models.User} "用户详情"
// @Router      /v1/my/info [get]
// @Security    JWT
func (h *MyHandler) Myinfo(c *gin.Context) {
	u, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, i18n.Errorf(c, "unauthorized, please login first"))
		return
	}
	var user models.User
	if e := h.GetDB().Preload("SystemRole").Preload("Tenants").First(&user, "id = ?", u.GetID()).Error; e != nil {
		handlers.Forbidden(c, i18n.Errorf(c, "forbidden, please login"))
		return
	}
	handlers.OK(c, user)
}

// MyAuthority 获取当前用户权限列表
// @Tags        User
// @Summary     获取当前用户权限列表
// @Description 获取当前用户权限列表
// @Accept      json
// @Produce     json
// @Success     200 {object} handlers.ResponseStruct{} "用户权限列表"
// @Router      /v1/my/auth [get]
// @Security    JWT
func (h *MyHandler) MyAuthority(c *gin.Context) {
	u, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, i18n.Errorf(c, "unauthorized, please login first"))
		return
	}

	auth := h.ModelCache().FlushUserAuthority(u)
	handlers.OK(c, auth)
}

// MyTenants 获取当前用户租户列表
// @Tags        User
// @Summary     获取当前用户租户列表
// @Description 获取当前用户租户列表
// @Accept      json
// @Produce     json
// @Success     200 {object} handlers.ResponseStruct{} "用户权限列表"
// @Router      /v1/my/tenants [get]
// @Security    JWT
func (h *MyHandler) MyTenants(c *gin.Context) {
	u, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, i18n.Errorf(c, "unauthorized, please login first"))
		return
	}
	tenants := []models.Tenant{}
	h.GetDB().
		Joins("tenant_user_rels on tenant_user_rels.tenant_id = tenants.id").
		Where("tenant_user_rels.user_id = ?", u.GetID()).Find(&tenants)
	handlers.OK(c, tenants)
}

// ResetPassword 重设密码
// @Tags        User
// @Summary     重设密码
// @Description 重设密码
// @Accept      json
// @Produce     json
// @Param       param body     resetPasswordForm         true "表单"
// @Success     200   {object} handlers.ResponseStruct{} ""
// @Router      /v1/my/reset_password [post]
// @Security    JWT
func (h *MyHandler) ResetPassword(c *gin.Context) {
	u, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, i18n.Errorf(c, "unauthorized, please login first"))
		return
	}
	cuser := models.User{}
	h.GetDB().First(&cuser, u.GetID())
	form := &resetPasswordForm{}
	c.BindJSON(form)

	if form.New1 != form.New2 {
		handlers.NotOK(c, i18n.Errorf(c, "the passwords entered twice are inconsistent"))
		return
	}

	if err := utils.ValidPassword(form.New1); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := utils.ValidatePassword(form.Origin, cuser.Password); err != nil {
		handlers.NotOK(c, i18n.Errorf(c, "origin password error"))
		return
	}

	pass, err := utils.MakePassword(form.New1)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cuser.Password = pass
	if err := h.GetDB().Save(&cuser).Error; err != nil {
		return
	}
	handlers.OK(c, nil)
}

type resetPasswordForm struct {
	Origin string `json:"origin"`
	New1   string `json:"new1"`
	New2   string `json:"new2"`
}
