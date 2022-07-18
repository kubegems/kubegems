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

package aaa

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/models"
)

type ContextUserGetter interface {
	GetContextUser(c *gin.Context) (models.CommonUserIface, bool)
}

type ContextUserSetter interface {
	SetContextUser(c *gin.Context, user models.CommonUserIface)
}
type ContextUserOperator interface {
	ContextUserGetter
	ContextUserSetter
}

type UserInfoHandler struct {
	ContextUserKey string
}

func NewUserInfoHandler() *UserInfoHandler {
	return &UserInfoHandler{
		ContextUserKey: "current_user",
	}
}

func (i *UserInfoHandler) SetContextUser(c *gin.Context, user models.CommonUserIface) {
	c.Set(i.ContextUserKey, user)
}

func (i *UserInfoHandler) GetContextUser(c *gin.Context) (models.CommonUserIface, bool) {
	user, exist := c.Get(i.ContextUserKey)
	if exist {
		return user.(models.CommonUserIface), true
	}
	return nil, false
}
