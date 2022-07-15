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
	"strings"

	"github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
)

type ManifestHandler struct {
	BaseHandler
	*ManifestProcessor
}

type Manifest struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Kind         string            `json:"kind"`
	Images       []string          `json:"images"`
	Creator      string            `json:"creator"`
	CreateAt     metav1.Time       `json:"createAt"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Ref          PathRef           `json:"ref"`
	IstioVersion string            `json:"istioVersion"`
}

// @Tags         ApplicationManifest
// @Summary      创建应用编排
// @Description  只创建一个空的应用没有内容文件
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      int                                     true  "tenaut id"
// @Param        project_id  path      int                                     true  "project id"
// @Param        body        body      Manifest                                true  "project meta"
// @Success      200         {object}  handlers.ResponseStruct{Data=Manifest}  "Application"
// @Router       /v1/tenant/{tenaut_id}/project/{project_id}/manifests [post]
// @Security     JWT
func (h *ManifestHandler) CreateManifest(req *restful.Request, resp *restful.Response) {
	manifest := &Manifest{}
	h.NoNameRefFunc(req, resp, manifest, func(ctx context.Context, ref PathRef) (interface{}, error) {
		if manifest.Name == "" {
			return nil, fmt.Errorf("empty manifest name")
		}

		ref.Name = manifest.Name
		ref.Name = strings.ToLower(ref.Name) // 小写
		if err := h.ManifestProcessor.Create(ctx, ref, *manifest); err != nil {
			return nil, err
		}
		return manifest, nil
	})
}

// @Tags         ApplicationManifest
// @Summary      应用编排详情
// @Description  应用编排详情
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      int                                     true  "tenaut id"
// @Param        project_id  path      int                                     true  "project id"
// @Param        name        path      string                                  true  "application name"
// @Success      200         {object}  handlers.ResponseStruct{Data=Manifest}  "Application"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name} [get]
// @Security     JWT
func (h *ManifestHandler) GetManifest(req *restful.Request, resp *restful.Response) {
	h.NamedRefFunc(req, resp, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		return h.Get(ctx, ref)
	})
}

// @Tags         ApplicationManifest
// @Summary      修改应用编排描述
// @Description  修改应用编排描述
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      int                                     true  "tenaut id"
// @Param        project_id  path      int                                     true  "project id"
// @Param        name        path      string                                  true  "application name"
// @Param        body        body      Manifest                                true  "project meta"
// @Success      200         {object}  handlers.ResponseStruct{Data=Manifest}  "Application"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name} [put]
// @Security     JWT
func (h *ManifestHandler) UpdateManifest(req *restful.Request, resp *restful.Response) {
	manifest := &Manifest{}
	h.NamedRefFunc(req, resp, manifest, func(ctx context.Context, ref PathRef) (interface{}, error) {
		if err := h.ManifestProcessor.Update(ctx, ref, *manifest); err != nil {
			return nil, err
		}
		return manifest, nil
	})
}

// @Tags         ApplicationManifest
// @Summary      应用编排列表
// @Description  应用编排列表
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      int                                       true  "tenaut id"
// @Param        project_id  path      int                                       true  "project id"
// @Success      200         {object}  handlers.ResponseStruct{Data=[]Manifest}  "Application"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/manifests [get]
// @Security     JWT
func (h *ManifestHandler) ListManifest(req *restful.Request, resp *restful.Response) {
	h.NoNameRefFunc(req, resp, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		manifestList, err := h.ManifestProcessor.List(ctx, ref, WithImages())
		if err != nil {
			return nil, err
		}
		// page
		searchnamefunc := func(i int) bool {
			return strings.Contains(manifestList[i].Name, req.QueryParameter("search"))
		}
		return handlers.NewPageDataFromContext(req, manifestList, searchnamefunc, nil), nil
	})
}

// @Tags         ApplicationManifest
// @Summary      删除创建应用编排
// @Description  删除创建应用编排以及git内容
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      int                                   true  "tenaut id"
// @Param        project_id  path      int                                   true  "project id"
// @Param        name        path      string                                true  "name"
// @Success      200         {object}  handlers.ResponseStruct{Data=string}  "Application"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name} [delete]
// @Security     JWT
func (h *ManifestHandler) DeleteManifest(req *restful.Request, resp *restful.Response) {
	h.NamedRefFunc(req, resp, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		if err := h.ManifestProcessor.Remove(ctx, ref); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}
