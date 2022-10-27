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
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
)

type UserHandler struct {
	base.BaseHandler
}

func (h *UserHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/user", h.CheckIsSysADMIN, h.ListUser)
	rg.POST("/user", h.CheckIsSysADMIN, h.PostUser)
	rg.GET("/user/:user_id", h.CheckIsSysADMIN, h.RetrieveUser)
	rg.PUT("/user/:user_id", h.CheckIsSysADMIN, h.PutUser)
	rg.PUT("/user", h.SelfUpdateInfo)
	rg.DELETE("/user/:user_id", h.CheckIsSysADMIN, h.DeleteUser)
	rg.GET("/user/:user_id/tenant", h.ListUserTenant)
	rg.POST("/user/:user_id/reset_password", h.CheckIsSysADMIN, h.ResetUserPassword)
	rg.GET("/user/_/environment/:environment_id", h.ListEnvironmentUser) // TODO: 严格来说，应该校验这些环境是否在用户当前的虚拟空间中
}
