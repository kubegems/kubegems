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

package sels

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
)

type SelsHandler struct {
	base.BaseHandler
}

func (h *SelsHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/sels/users", h.UserSels)
	rg.GET("/sels/tenants", h.TenantSels)
	rg.GET("/sels/projects", h.ProjectSels)
	rg.GET("/sels/environments", h.EnvironmentSels)
	rg.GET("/sels/applications", h.ApplicationSels)
}
