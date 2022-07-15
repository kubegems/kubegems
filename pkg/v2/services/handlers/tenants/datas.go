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

package tenanthandler

import (
	"kubegems.io/kubegems/pkg/v2/models"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
)

type TenantCreateResp struct {
	handlers.RespBase
	Data models.TenantSimple `json:"data"`
}

type TenantCommonResp struct {
	handlers.RespBase
	Data models.TenantCommon `json:"data"`
}

type TenantListResp struct {
	handlers.ListBase
	Data []models.TenantSimple `json:"list"`
}

type ProjectListResp struct {
	handlers.ListBase
	Data []models.Project `json:"list"`
}

type EnvironmentListResp struct {
	handlers.ListBase
	Data []models.EnvironmentCommon `json:"list"`
}

type EnvironmentResp struct {
	handlers.RespBase
	Data models.Environment `json:"data"`
}

type ProjectResp struct {
	handlers.RespBase
	Data models.Project `json:"data"`
}

type UserSimpleListResp struct {
	handlers.ListBase
	Data []models.UserSimple `json:"list"`
}

type TenantUserCreateForm struct {
	Tenant string `json:"tenant" validate:"required"`
	User   string `json:"user" validate:"required"`
	Role   string `json:"role" validate:"required"`
}

type TenantUserCreateResp struct {
	handlers.RespBase
	Data TenantUserCreateForm `json:"data"`
}

type ProjectCreateForm struct {
	Name          string `json:"name" validate:"required"`
	Remark        string `json:"remark" validate:"required"`
	ResourceQuota string `json:"quota" validate:"json"`
}

type EnvironmentResourceResp struct {
	handlers.RespBase
	Data models.EnvironmentResource `json:"data"`
}

type EnvironmentUserRelsResp struct {
	handlers.RespBase
	Data models.EnvironmentUserRels `json:"data"`
}

type EnvironmentCreateForm struct {
	Name          string `json:"name,omitempty" validate:"required"`
	Namespace     string `json:"namespace,omitempty" validate:"required"`
	Remark        string `json:"remark,omitempty" validate:"required"`
	MetaType      string `json:"metaType,omitempty" validate:"required"`
	DeletePolicy  string `json:"deletePolicy,omitempty" validate:"required"`
	Cluster       string `json:"cluster,omitempty" validate:"required"`
	Project       string `json:"project,omitempty" validate:"required"`
	ResourceQuota string `json:"resourceQuota,omitempty" validate:"required,json"`
	LimitRange    string `json:"limitRange,omitempty" validate:"required,json"`
	ProjectID     uint   `json:"projectID,omitempty"`
	ClusterID     uint   `json:"clusterID,omitempty"`
	CreatorID     uint   `json:"creatorID,omitempty"`
}
