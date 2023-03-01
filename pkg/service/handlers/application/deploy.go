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

package application

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"kubegems.io/kubegems/pkg/apis/application"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/git"
)

const StatusNoArgoApp = "NoArgoApp"

const (
	// labels
	LabelTenant      = gems.LabelTenant
	LabelProject     = gems.LabelProject
	LabelApplication = gems.LabelApplication
	LabelEnvironment = gems.LabelEnvironment

	// application label
	LabelKeyFrom           = application.LabelFrom // 区分是从 appstore 还是从应用 app 部署的argo
	LabelValueFromApp      = application.LabelValueFromApp
	LabelValueFromAppStore = application.LabelValueFromAppStore

	// annotations
	AnnotationKeyCreator = application.AnnotationCreator   // 创建人,仅用于当前部署实时更新，从kustomize部署的历史需要从gitcommit取得
	AnnotationRef        = application.AnnotationRef       // 标志这个资源所属的项目环境，避免使用过多label造成干扰
	AnnotationCluster    = application.AnnotationCluster   // 标志这个资源所属集群
	AnnotationNamespace  = application.AnnotationNamespace // 标志这个资源所属namespace
)

type DeploiedManifest struct {
	Manifest `json:",inline"`
	Runtime  ManifestRuntime `json:"runtime"`
}

type ManifestRuntime struct {
	Status       string            `json:"status"` // 运行时状态
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Kind         string            `json:"kind"`         // 运行时负载类型
	WorkloadName string            `json:"workloadName"` // 运行时
	Images       []string          `json:"images"`       // 运行时镜像
	Message      string            `json:"message"`      // 运行时消息提示
	CreateAt     metav1.Time       `json:"createAt"`     // 运行时创建时间
	Creator      string            `json:"creator"`      // 运行时创建人
	Raw          interface{}       `json:"raw"`          // 运行时
	DeployStatus string            `json:"deployStatus"` // 异步部署的状态，取最新一个
	IstioVersion string            `json:"istioVersion"` // 运行时的 istio version
}

type ManifestDeploy struct {
	Cluster   string
	Namespace string
	Name      string
	Contents  []unstructured.Unstructured
}

func MustNewApplicationDeployHandler(gitoptions *git.Options, argocli *argo.Client, commonbase base.BaseHandler) *ApplicationHandler {
	provider, err := git.NewProvider(gitoptions)
	if err != nil {
		panic(err)
	}
	database := commonbase.GetDataBase()
	agents := commonbase.GetAgents()
	redis := commonbase.GetRedis()

	base := BaseHandler{
		BaseHandler: commonbase,

		dbcahce: &Cache{},
	}

	h := &ApplicationHandler{
		Agents:      commonbase.GetAgents(),
		BaseHandler: base,
		ArgoCD:      argocli,
		Manifest: ManifestHandler{
			BaseHandler:       base,
			ManifestProcessor: &ManifestProcessor{GitProvider: provider},
		},
		Task:                 NewTaskHandler(base),
		ApplicationProcessor: NewApplicationProcessor(database, provider, argocli, redis, agents),
	}
	return h
}

// @Tags        Application
// @Summary     应用列表
// @Description 应用列表
// @Accept      json
// @Produce     json
// @Param       tenant_id      path     int                                              true "tenaut id"
// @Param       project_id     path     int                                              true "project id"
// @Param       environment_id path     int                                              true "environment_id"
// @Success     200            {object} handlers.ResponseStruct{Data=[]DeploiedManifest} "Application"
// @Router      /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications [get]
// @Security    JWT
func (h *ApplicationHandler) List(c *gin.Context) {
	h.NoNameRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		dm, err := h.ApplicationProcessor.List(ctx, ref)
		if err != nil {
			return nil, err
		}
		// 分页
		namefunc := func(i DeploiedManifest) string {
			return i.Name
		}
		timefunc := func(i DeploiedManifest) time.Time {
			return i.CreateAt.Time
		}
		paged := handlers.NewPageDataFromContext(c, dm, namefunc, timefunc)
		return paged, nil
	})
}

// @Tags        Application
// @Summary     部署应用
// @Description 应用部署
// @Accept      json
// @Produce     json
// @Param       tenant_id      path     int                                            true "tenaut id"
// @Param       project_id     path     int                                            true "project id"
// @Param       environment_id path     int                                            true "environment_id"
// @Success     200            {object} handlers.ResponseStruct{Data=DeploiedManifest} "Application"
// @Router      /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications [post]
// @Security    JWT
func (h *ApplicationHandler) Create(c *gin.Context) {
	body := &DeploiedManifest{}
	h.NoNameRefFunc(c, body, func(ctx context.Context, ref PathRef) (interface{}, error) {
		if body.Name == "" {
			return nil, fmt.Errorf("empty manifest name")
		}
		ref.Name = body.Name
		// audit
		h.SetAuditData(c, "创建", "应用", ref.Name)
		if err := h.ApplicationProcessor.Create(ctx, ref); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

// @Tags        Application
// @Summary     批量部署应用
// @Description 批量部署应用
// @Accept      json
// @Produce     json
// @Param       tenant_id      path     int                                            true "tenaut id"
// @Param       project_id     path     int                                            true "project id"
// @Param       environment_id path     int                                            true "environment_id"
// @Param       body           body     []DeploiedManifest                             true "body"
// @Success     200            {object} handlers.ResponseStruct{Data=DeploiedManifest} "Application"
// @Router      /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications-batch [post]
// @Security    JWT
func (h *ApplicationHandler) CreateBatch(c *gin.Context) {
	body := []DeploiedManifest{}
	h.NoNameRefFunc(c, &body, func(ctx context.Context, ref PathRef) (interface{}, error) {
		// audit
		names := make([]string, 0, len(body))
		for _, v := range body {
			names = append(names, v.Name)
		}
		h.SetAuditData(c, "批量创建", "应用", ref.Name)
		if err := h.ApplicationProcessor.CreateBatch(ctx, ref, names); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

// @Tags        Application
// @Summary     应用部署
// @Description 应用部署
// @Accept      json
// @Produce     json
// @Param       tenant_id      path     int                                            true "tenaut id"
// @Param       project_id     path     int                                            true "project id"
// @Param       environment_id path     int                                            true "environment_id"
// @Param       name           path     string                                         true "application name"
// @Success     200            {object} handlers.ResponseStruct{Data=DeploiedManifest} "Application"
// @Router      /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name} [get]
// @Security    JWT
func (h *ApplicationHandler) Get(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		return h.ApplicationProcessor.Get(ctx, ref)
	})
}

// @Tags        Application
// @Summary     删除应用
// @Description 删除应用
// @Accept      json
// @Produce     json
// @Param       tenant_id      path     int                                  true "tenaut id"
// @Param       project_id     path     int                                  true "project id"
// @Param       environment_id path     int                                  true "environment_id"
// @Param       name           path     string                               true "application name"
// @Success     200            {object} handlers.ResponseStruct{Data=string} "Application"
// @Router      /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name} [delete]
// @Security    JWT
func (h *ApplicationHandler) Remove(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		// audit
		h.SetAuditData(c, "删除", "应用", ref.Name)
		if err := h.ApplicationProcessor.Remove(ctx, ref); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}
