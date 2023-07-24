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
	"context"
	"strings"

	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils"

	"github.com/gin-gonic/gin"
)

var (
	SearchFields   = []string{"Username", "Email"}
	FilterFields   = []string{"Username"}
	PreloadFields  = []string{"Tenants", "SystemRole"}
	OrderFields    = []string{"Username", "ID"}
	ModelName      = "User"
	PrimaryKeyName = "user_id"
)

// ListUser 列表 User
//	@Tags			User
//	@Summary		User列表
//	@Description	User列表
//	@Accept			json
//	@Produce		json
//	@Param			Username	query		string																false	"Username"
//	@Param			preload		query		string																false	"choices Tenants,SystemRole"
//	@Param			page		query		int																	false	"page"
//	@Param			size		query		int																	false	"page"
//	@Param			search		query		string																false	"search in (Username,Email)"
//	@Success		200			{object}	handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}}	"User"
//	@Router			/v1/user [get]
//	@Security		JWT
func (h *UserHandler) ListUser(c *gin.Context) {
	var list []models.User
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "User",
		PreloadFields: PreloadFields,
		SearchFields:  SearchFields,
		SortFields:    OrderFields,
	}
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveUser User详情
//	@Tags			User
//	@Summary		User详情
//	@Description	get User详情
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		uint										true	"user_id"
//	@Success		200		{object}	handlers.ResponseStruct{Data=models.User}	"User"
//	@Router			/v1/user/{user_id} [get]
//	@Security		JWT
func (h *UserHandler) RetrieveUser(c *gin.Context) {
	var obj models.User
	if err := h.GetDB().WithContext(c.Request.Context()).First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// PostUser 创建User
//	@Tags			User
//	@Summary		创建User
//	@Description	创建User
//	@Accept			json
//	@Produce		json
//	@Param			param	body		models.UserCreate								true	"表单"
//	@Success		200		{object}	handlers.ResponseStruct{Data=models.UserCreate}	"User"
//	@Router			/v1/user [post]
//	@Security		JWT
func (h *UserHandler) PostUser(c *gin.Context) {
	var obj models.UserCreate
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	pass, err := utils.MakePassword(obj.Password)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	truePtr := true
	user := &models.User{
		Username:     obj.Username,
		Password:     pass,
		Email:        obj.Email,
		SystemRoleID: 2,
		IsActive:     &truePtr,
	}
	if err := h.GetDB().WithContext(c.Request.Context()).Create(&user).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	action := i18n.Sprintf(context.TODO(), "create")
	module := i18n.Sprintf(context.TODO(), "account")
	h.SetAuditData(c, action, module, user.Username)
	handlers.Created(c, user)
}

// UpdateSelfInfo  update user self infomation
//	@Tags			User
//	@Summary		self update information
//	@Description	self update
//	@Accept			json
//	@Produce		json
//	@Param			param	body		models.User									true	"表单"
//	@Success		200		{object}	handlers.ResponseStruct{Data=models.User}	"User"
//	@Router			/v1/user [put]
//	@Security		JWT
func (h *UserHandler) SelfUpdateInfo(c *gin.Context) {
	h.PutUser(c)
}

// PutUser 修改User
//	@Tags			User
//	@Summary		修改User
//	@Description	修改User，目前只能修改Email、Phone
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		uint										true	"user_id"
//	@Param			param	body		models.User									true	"表单"
//	@Success		200		{object}	handlers.ResponseStruct{Data=models.User}	"User"
//	@Router			/v1/user/{user_id} [put]
//	@Security		JWT
func (h *UserHandler) PutUser(c *gin.Context) {
	var (
		selfupdate bool
		userId     uint
	)
	userId = utils.ToUint(c.Param(PrimaryKeyName))
	if userId == 0 {
		selfupdate = true
		user, exist := h.GetContextUser(c)
		if !exist {
			handlers.NotOK(c, i18n.Error(c, "can't modify current user's infomation"))
			return
		}
		userId = user.GetID()
	}
	var oldUser, newUser models.User
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).First(&oldUser, userId).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := c.BindJSON(&newUser); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if !selfupdate && newUser.ID != userId {
		handlers.NotOK(c, i18n.Errorf(c, "URL parameter mismatched with body"))
		return
	}

	oldUser.Email = newUser.Email
	oldUser.Phone = newUser.Phone

	if err := h.GetDB().WithContext(ctx).Save(&oldUser).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	if !selfupdate {
		action := i18n.Sprintf(context.TODO(), "update")
		module := i18n.Sprintf(context.TODO(), "account")
		h.SetAuditData(c, action, module, oldUser.Username)
	}

	handlers.OK(c, oldUser)
}

// DeleteUser 删除 User
//	@Tags			User
//	@Summary		删除 User
//	@Description	删除 User
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		uint					true	"user_id"
//	@Success		204		{object}	handlers.ResponseStruct	"resp"
//	@Router			/v1/user/{user_id} [delete]
//	@Security		JWT
func (h *UserHandler) DeleteUser(c *gin.Context) {
	var obj models.User
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, nil)
		return
	}
	if err := h.GetDB().WithContext(ctx).Delete(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "account")
	h.SetAuditData(c, action, module, obj.Username)
	handlers.NoContent(c, nil)
}

// ListUserTenant 获取属于User的 Tenant 列表
//	@Tags			User
//	@Summary		获取属于 User 的 Tenant 列表
//	@Description	获取属于 User 的 Tenant 列表
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		uint																	true	"user_id"
//	@Param			preload	query		string																	false	"choices ResourceQuotas,Users,Projects"
//	@Param			page	query		int																		false	"page"
//	@Param			size	query		int																		false	"page"
//	@Param			search	query		string																	false	"search in (TenantName,Remark)"
//	@Success		200		{object}	handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Tenant}}	"models.Tenant"
//	@Router			/v1/user/{user_id}/tenant [get]
//	@Security		JWT
func (h *UserHandler) ListUserTenant(c *gin.Context) {
	var list []models.Tenant

	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	query.Order = "TenantName"
	cond := &handlers.PageQueryCond{
		Model:         "Tenant",
		PreloadFields: []string{"ResourceQuotas", "Users", "Projects"},
		SearchFields:  []string{"TenantName", "Remark"},
		Join:          handlers.Args("join tenant_user_rels on tenant_user_rels.tenant_id = tenants.id"),
		Where:         []*handlers.QArgs{handlers.Args("tenant_user_rels.user_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

type ResetPasswordReq struct {
	Password string `json:"password"`
}

// ResetUserPassword 重置用户密码
//	@Tags			User
//	@Summary		重置用户密码
//	@Description	重置用户密码
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		uint												true	"user_id"
//	@Success		200		{object}	handlers.ResponseStruct{Data=resetPasswordResult}	"data"
//	@Router			/v1/user/{user_id}/reset_password [post]
//	@Security		JWT
func (h *UserHandler) ResetUserPassword(c *gin.Context) {
	var user models.User
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).First(&user, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	resetPasswordReq := &ResetPasswordReq{}
	c.BindJSON(resetPasswordReq)
	var newPassowrd string
	if resetPasswordReq.Password != "" {
		if err := utils.ValidPassword(resetPasswordReq.Password); err != nil {
			handlers.NotOK(c, err)
			return
		}
		newPassowrd = resetPasswordReq.Password
	} else {
		newPassowrd = utils.GeneratePassword()
	}
	encryptPassword, err := utils.MakePassword(newPassowrd)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	user.Password = encryptPassword
	if err := h.GetDB().WithContext(ctx).Save(&user).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, &resetPasswordResult{Password: newPassowrd})
}

// ListEnvironmentUser 获取多个环境的用户列表
//	@Tags			User
//	@Summary		获取多个环境的用户列表
//	@Description	获取多个环境的用户列表
//	@Accept			json
//	@Produce		json
//	@Param			environment_id	path		uint																true	"环境id，中间以逗号隔开"
//	@Param			page			query		int																	false	"page"
//	@Param			size			query		int																	false	"page"
//	@Param			search			query		string																false	"search in (Username,Email)"
//	@Success		200				{object}	handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}}	"models.User"
//	@Router			/v1/user/_/environment/{environment_id} [get]
//	@Security		JWT
func (h *UserHandler) ListEnvironmentUser(c *gin.Context) {
	var list []models.User
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:        "User",
		SearchFields: []string{"Username", "Email"},
		Select:       handlers.Args("users.*, environment_user_rels.role"),
		Join:         handlers.Args("join environment_user_rels on environment_user_rels.user_id = users.id"),
		Where:        []*handlers.QArgs{handlers.Args("environment_user_rels.environment_id in ?", strings.Split(c.Param("environment_id"), ","))},
	}
	total, page, size, err := query.PageList(h.GetDB().WithContext(c.Request.Context()), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

type resetPasswordResult struct {
	Password string `json:"password"`
}
