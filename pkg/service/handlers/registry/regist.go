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

package registryhandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
)

type RegistryHandler struct {
	base.BaseHandler
}

func (h *RegistryHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/registry", h.CheckIsSysADMIN, h.ListRegistry)
	rg.GET("/registry/:registry_id", h.CheckIsSysADMIN, h.RetrieveRegistry)
	rg.PUT("/registry/:registry_id", h.CheckIsSysADMIN, h.PutRegistry)
	rg.DELETE("/registry/:registry_id", h.CheckIsSysADMIN, h.DeleteRegistry)

	rg.POST("/project/:project_id/registry", h.CheckByProjectID, h.PostProjectRegistry)
	rg.GET("/project/:project_id/registry", h.CheckByProjectID, h.ListProjectRegistry)
	rg.GET("/project/:project_id/registry/:registry_id", h.CheckByProjectID, h.RetrieveProjectRegistry)
	rg.PUT("/project/:project_id/registry/:registry_id", h.CheckByProjectID, h.PutProjectRegistry)
	rg.PATCH("/project/:project_id/registry/:registry_id", h.CheckByProjectID, h.SetDefaultProjectRegistry)
	rg.DELETE("/project/:project_id/registry/:registry_id", h.CheckByProjectID, h.DeleteProjectRegistry)
}
