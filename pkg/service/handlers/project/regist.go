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

package projecthandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
)

type ProjectHandler struct {
	base.BaseHandler
}

func (h *ProjectHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/project", h.CheckIsSysADMIN, h.ListProject)
	rg.GET("/project/:project_id", h.CheckByProjectID, h.RetrieveProject)
	rg.PUT("/project/:project_id",
		h.CheckByProjectID, h.PutProject)
	rg.DELETE("/project/:project_id",
		h.CheckByProjectID, h.DeleteProject)

	rg.POST("/project/:project_id/action/networkisolate",
		h.CheckByProjectID, h.ProjectSwitch)

	rg.GET("/project/:project_id/user", h.CheckByProjectID, h.ListProjectUser)
	rg.GET("/project/:project_id/user/:user_id", h.CheckByProjectID, h.RetrieveProjectUser)
	rg.POST("/project/:project_id/user",
		h.CheckByProjectID, h.PostProjectUser)
	rg.PUT("/project/:project_id/user/:user_id",
		h.CheckByProjectID, h.PutProjectUser)
	rg.DELETE("/project/:project_id/user/:user_id",
		h.CheckByProjectID, h.DeleteProjectUser)

	rg.GET("/project/:project_id/environment", h.CheckByProjectID, h.ListProjectEnvironment)
	rg.GET("/project/:project_id/environment/:environment_id", h.CheckByEnvironmentID, h.RetrieveProjectEnvironment)
	rg.POST("/project/:project_id/environment",
		h.CheckByProjectID, h.PostProjectEnvironment)

	rg.GET("/project/:project_id/statistics", h.CheckByProjectID, h.ProjectStatistics)
	rg.GET("/project/:project_id/none_resource_statistics", h.CheckByProjectID, h.ProjectNoneResourceStatistics)
	rg.GET("/project/:project_id/quota", h.CheckByProjectID, h.GetProjectResourceQuota)
	rg.GET("/project/:project_id/quotas", h.CheckIsSysADMIN, h.GetProjectListResourceQuotas)
	rg.GET("/project/:project_id/agg_environment", h.CheckByProjectID, h.ProjectEnvironments)
	rg.GET("/project/:project_id/resources", h.CheckByProjectID, h.GetProjectResource)

	rg.GET("/project/:project_id/environment/:environment_id/statistics", h.CheckByProjectID, h.EnvironmentStatistics)
	rg.GET("/project/:project_id/environment/:environment_id/quota", h.CheckByProjectID, h.GetEnvironmentResourceQuota)
	rg.GET("/project/:project_id/environment/:environment_id/quotas", h.CheckByProjectID, h.GetEnvironmentResourceQuotas)

	rg.GET("/tenant/:tenant_id/projectquotas", h.CheckByTenantID, h.TenantProjectListResourceQuotas)
}
