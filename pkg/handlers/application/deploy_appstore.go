package application

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/handlers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type HelmManifest struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Kind         string      `json:"kind"`
	Creator      string      `json:"creator"`
	CreateAt     metav1.Time `json:"createAt"`
	RepoURL      string      `json:"repoURL"`      // 仓库index地址
	Chart        string      `json:"chart"`        // chart名称
	ChartVersion string      `json:"chartVersion"` // chart版本
	Values       string      `json:"values"`
}

type DeploiedHelm struct {
	HelmManifest `json:",inline"`
	Runtime      ManifestRuntime `json:"runtime"`
}

// @Tags Application
// @Summary 应用商店应用列表
// @Description 应用商店应用列表
// @Accept json
// @Produce json
// @Param tenant_id      path  int    true "tenaut id"
// @Param project_id     path  int    true "project id"
// @Param environment_id path  int    true "environment_id"
// @Success 200 {object} handlers.ResponseStruct{Data=[]DeploiedManifest} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/appstoreapplications [get]
// @Security JWT
func (h *ApplicationHandler) ListAppstoreApp(c *gin.Context) {
	h.NoNameRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		// list argo apps
		applist, err := h.ArgoCD.ListArgoApp(ctx, labels.Set{
			ArgoLabelKeyFrom:     ArgoLabelValueFromAppStore,
			ArgoLabelTenant:      ref.Tenant,
			ArgoLabelProject:     ref.Project,
			ArgoLabelEnvironment: ref.Env,
		}.AsSelector())
		if err != nil {
			return nil, err
		}

		list := []*DeploiedHelm{}
		for _, app := range applist.Items {
			deploied := CompleteDeploiedManifestRuntime(&app, &DeploiedManifest{})
			deploiedhelm := &DeploiedHelm{
				HelmManifest: HelmManifest{
					Name:         deploied.Name,
					Kind:         deploied.Runtime.Kind,
					Description:  deploied.Description,
					Creator:      deploied.Creator,
					CreateAt:     deploied.CreateAt,
					RepoURL:      app.Spec.Source.RepoURL,
					ChartVersion: app.Spec.Source.TargetRevision,
					Chart:        app.Spec.Source.Chart,
				},
				Runtime: deploied.Runtime,
			}
			if helm := app.Spec.Source.Helm; helm != nil {
				deploiedhelm.Values = app.Spec.Source.Helm.Values
			}
			list = append(list, deploiedhelm)
		}
		// 分页
		searchnamefunc := func(i int) bool {
			return strings.Contains(list[i].Name, c.Query("search"))
		}
		// 排序
		sortbycreatfunc := func(i, j int) bool {
			return list[i].CreateAt.After(list[j].CreateAt.Time)
		}
		return handlers.NewPageDataFromContext(c, list, searchnamefunc, sortbycreatfunc), nil
	})
}

// @Tags Application
// @Summary 应用商店应用
// @Description 应用商店应用
// @Accept json
// @Produce json
// @Param tenant_id      path  int    true "tenaut id"
// @Param project_id     path  int    true "project id"
// @Param environment_id path  int    true "environment_id"
// @Param name			 path  string	true "application name"
// @Success 200 {object} handlers.ResponseStruct{Data=DeploiedManifest} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/appstoreapplications/{name} [get]
// @Security JWT
func (h *ApplicationHandler) GetAppstoreApp(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		deploied := &DeploiedManifest{Manifest: Manifest{}}
		app, err := h.ArgoCD.GetArgoApp(ctx, ref.FullName())
		if err != nil {
			return nil, err
		}
		CompleteDeploiedManifestRuntime(app, deploied)
		return deploied, nil
	})
}

// @Tags Application
// @Summary 应用商店部署
// @Description 应用商店部署
// @Accept json
// @Produce json
// @Param tenant_id      path  int    true "tenaut id"
// @Param project_id     path  int    true "project id"
// @Param environment_id path  int    true "environment_id"
// @Param body			 body  AppStoreDeployForm true "chart 部署信息"
// @Success 200 {object} handlers.ResponseStruct{Data=[]DeploiedManifest} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/appstoreapplications [post]
// @Security JWT
func (h *ApplicationHandler) CreateAppstoreApp(c *gin.Context) {
	body := AppStoreDeployForm{}
	h.NoNameRefFunc(c, &body, func(ctx context.Context, ref PathRef) (interface{}, error) {
		// ref 中没有name
		ref.Name = strings.TrimSpace(body.Name)
		ref.Name = strings.ToLower(ref.Name) // 小写

		// audit
		h.SetAuditData(c, "部署", "应用商店应用", ref.Name)
		argoapp, err := h.ApplicationProcessor.deployHelmApplication(ctx, ref, body)
		if err != nil {
			return nil, err
		}
		return argoapp, nil
	})
}

// @Tags Application
// @Summary 应用商店应用列表
// @Description 应用商店应用列表
// @Accept json
// @Produce json
// @Param tenant_id      path  int    true "tenaut id"
// @Param project_id     path  int    true "project id"
// @Param environment_id path  int    true "environment_id"
// @Param name			 path  string	true "application name"
// @Success 200 {object} handlers.ResponseStruct{Data=[]DeploiedManifest} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/appstoreapplications [delete]
// @Security JWT
func (h *ApplicationHandler) RemoveAppstoreApp(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		// audit
		h.SetAuditData(c, "删除", "应用商店应用", ref.Name)

		if err := h.ApplicationProcessor.Remove(ctx, ref); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}
