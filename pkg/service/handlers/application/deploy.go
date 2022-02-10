package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	gemlabels "kubegems.io/pkg/labels"
	"kubegems.io/pkg/server/define"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/git"
)

const StatusNoArgoApp = "NoArgoApp"

const (
	// 应用common label
	CommonLabelApplication = gemlabels.LabelApplication

	// argo labels
	ArgoLabelTenant            = gemlabels.LabelTenant
	ArgoLabelProject           = gemlabels.LabelProject
	ArgoLabelApplication       = gemlabels.LabelApplication
	ArgoLabelEnvironment       = gemlabels.LabelEnvironment
	ArgoLabelKeyFrom           = gemlabels.ArgoLabelKeyFrom // 区分是从 appstore 还是从应用 app 部署的argo
	ArgoLabelValueFromApp      = gemlabels.ArgoLabelValueFromApp
	ArgoLabelValueFromAppStore = gemlabels.ArgoLabelValueFromAppStore
	// argo annotations
	ArgoAnnotationKeyCreator = gemlabels.ArgoLabelKeyCreator   // 创建人,仅用于当前部署实时更新，从kustomize部署的历史需要从gitcommit取得
	AnnotationRef            = "gems.cloudminds.com/ref"       // 标志这个资源所属的项目环境，避免使用过多label造成干扰
	AnnotationCluster        = "gems.cloudminds.com/cluster"   // 标志这个资源所属集群
	AnnotationNamespace      = "gems.cloudminds.com/namespace" // 标志这个资源所属namespace
)

type DeploiedManifest struct {
	Manifest `json:",inline"`
	Runtime  ManifestRuntime `json:"runtime"`
}

type ManifestRuntime struct {
	Status       string      `json:"status"`       // 运行时状态
	Kind         string      `json:"kind"`         // 运行时负载类型
	WorkloadName string      `json:"workloadName"` // 运行时
	Images       []string    `json:"images"`       // 运行时镜像
	Message      string      `json:"message"`      // 运行时消息提示
	CreateAt     metav1.Time `json:"createAt"`     // 运行时创建时间
	Creator      string      `json:"creator"`      // 运行时创建人
	Raw          interface{} `json:"raw"`          // 运行时
	DeployStatus string      `json:"deployStatus"` // 异步部署的状态，取最新一个
	IstioVersion string      `json:"istioVersion"` // 运行时的 istio version
}

type ManifestDeploy struct {
	Cluster   string
	Namespace string
	Name      string
	Contents  []unstructured.Unstructured
}

func MustNewApplicationDeployHandler(server define.ServerInterface) *ApplicationHandler {
	opts := server.GetOptions().Git
	provider, err := git.NewProvider(opts)
	if err != nil {
		panic(err)
	}

	base := BaseHandler{
		ServerInterface: server,
		dbcahce:         &Cache{},
	}
	database := server.GetDataBase()
	argocli := server.GetArgocdClient()
	redis := server.GetRedis()

	h := &ApplicationHandler{
		BaseHandler: base,
		ArgoCD:      argocli,
		Manifest: ManifestHandler{
			BaseHandler:       base,
			ManifestProcessor: &ManifestProcessor{GitProvider: provider},
		},
		Task:                 NewTaskHandler(base),
		ApplicationProcessor: NewApplicationProcessor(database, provider, argocli, redis, argocli.AgentsCli),
	}
	return h
}

// @Tags Application
// @Summary 应用列表
// @Description 应用列表
// @Accept json
// @Produce json
// @Param tenant_id      path  int    true "tenaut id"
// @Param project_id     path  int    true "project id"
// @Param environment_id path  int    true "environment_id"
// @Success 200 {object} handlers.ResponseStruct{Data=[]DeploiedManifest} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications [get]
// @Security JWT
func (h *ApplicationHandler) List(c *gin.Context) {
	h.NoNameRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		dm, err := h.ApplicationProcessor.List(ctx, ref)
		if err != nil {
			return nil, err
		}
		// 分页
		searchnamefunc := func(i int) bool {
			return strings.Contains(dm[i].Name, c.Query("search"))
		}
		paged := handlers.NewPageDataFromContext(c, dm, searchnamefunc, nil)
		return paged, nil
	})
}

// @Tags Application
// @Summary 部署应用
// @Description 应用部署
// @Accept json
// @Produce json
// @Param tenant_id      path  int    true "tenaut id"
// @Param project_id     path  int    true "project id"
// @Param environment_id path  int    true "environment_id"
// @Success 200 {object} handlers.ResponseStruct{Data=DeploiedManifest} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications [post]
// @Security JWT
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

// @Tags Application
// @Summary 应用部署
// @Description 应用部署
// @Accept json
// @Produce json
// @Param tenant_id      path  int    true "tenaut id"
// @Param project_id     path  int    true "project id"
// @Param environment_id path  int    true "environment_id"
// @Param name			 path  string	true "application name"
// @Success 200 {object} handlers.ResponseStruct{Data=DeploiedManifest} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name} [get]
// @Security JWT
func (h *ApplicationHandler) Get(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		return h.ApplicationProcessor.Get(ctx, ref)
	})
}

// @Tags Application
// @Summary 删除应用
// @Description 删除应用
// @Accept json
// @Produce json
// @Param tenant_id      path  int    true "tenaut id"
// @Param project_id     path  int    true "project id"
// @Param environment_id path  int    true "environment_id"
// @Param name			 path  string	true "application name"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name} [delete]
// @Security JWT
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
