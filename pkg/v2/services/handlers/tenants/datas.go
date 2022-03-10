package tenanthandler

import (
	"kubegems.io/pkg/v2/models"
	"kubegems.io/pkg/v2/services/handlers"
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
