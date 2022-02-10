package application

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/utils/argo"
)

type ApplicationHandler struct {
	BaseHandler
	ApplicationProcessor *ApplicationProcessor
	Manifest             ManifestHandler
	Task                 *TaskHandler
	// self cluster cli
	ArgoCD *argo.Client
}

// nolint: lll,funlen
func (h *ApplicationHandler) RegistRouter(rg *gin.RouterGroup) error {
	deploy := h
	manifest := h.Manifest
	// 应用编排
	rg.GET("/tenant/_/project/_/manifests", manifest.ListManifestAdmin)
	rg.GET("/tenant/:tenant_id/project/:project_id/manifests", h.CheckByProjectID, manifest.ListManifest)
	rg.POST("/tenant/:tenant_id/project/:project_id/manifests", h.CheckCanDeployEnvironment, manifest.CreateManifest)
	rg.GET("/tenant/:tenant_id/project/:project_id/manifests/:name", h.CheckByProjectID, manifest.GetManifest)
	rg.PUT("/tenant/:tenant_id/project/:project_id/manifests/:name", h.CheckCanDeployEnvironment, manifest.UpdateManifest)
	rg.DELETE("/tenant/:tenant_id/project/:project_id/manifests/:name", h.CheckCanDeployEnvironment, manifest.RemoveManifest)
	// 编排文件
	rg.GET("/tenant/:tenant_id/project/:project_id/manifests/:name/files", h.CheckByProjectID, manifest.ListFiles)
	rg.PUT("/tenant/:tenant_id/project/:project_id/manifests/:name/files/:filename", h.CheckByProjectID, manifest.PutFile)
	rg.PUT("/tenant/:tenant_id/project/:project_id/manifests/:name/files", h.CheckByProjectID, manifest.PutFiles)
	rg.DELETE("/tenant/:tenant_id/project/:project_id/manifests/:name/files/:filename", h.CheckByProjectID, manifest.RemoveFile)
	// 编排文件git相关
	rg.GET("/tenant/:tenant_id/project/:project_id/manifests/:name/gitlog", h.CheckByProjectID, manifest.GitLog)
	rg.GET("/tenant/:tenant_id/project/:project_id/manifests/:name/gitdiff", h.CheckByProjectID, manifest.GitDiff)
	rg.POST("/tenant/:tenant_id/project/:project_id/manifests/:name/gitrevert", h.CheckByProjectID, manifest.GitRevert)
	rg.POST("/tenant/:tenant_id/project/:project_id/manifests/:name/gitpull", h.CheckByProjectID, manifest.GitPull)
	// 编排内的自动补全
	rg.GET("/tenant/:tenant_id/project/:project_id/manifests/:name/metas", h.CheckByProjectID, manifest.Metas)

	// 编排作为store
	rg.GET("/tenant/:tenant_id/project/:project_id/manifests/:name/resources/:group/:version/:kind", h.CheckByProjectID, manifest.ListResource)
	rg.GET("/tenant/:tenant_id/project/:project_id/manifests/:name/resources/:group/:version/:kind/:resourcename", h.CheckByProjectID, manifest.GetResource)
	rg.POST("/tenant/:tenant_id/project/:project_id/manifests/:name/resources/:group/:version/:kind/:resourcename", h.CheckByProjectID, manifest.CreateResource)
	rg.PUT("/tenant/:tenant_id/project/:project_id/manifests/:name/resources/:group/:version/:kind/:resourcename", h.CheckByProjectID, manifest.UpdateResource)
	rg.DELETE("/tenant/:tenant_id/project/:project_id/manifests/:name/resources/:group/:version/:kind/:resourcename", h.CheckByProjectID, manifest.DeleteResource)

	// Argo CD相关操作
	rg.GET("/tenant/:tenant_id/project/:project_id/manifests/:name/argohistory", h.CheckByProjectID, deploy.Argohistory)
	rg.GET("/tenant/:tenant_id/project/:project_id/manifests/:name/imagehistory", h.CheckByProjectID, deploy.ImageHistory)

	// 应用商店部署
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/appstoreapplications", h.CheckByEnvironmentID, deploy.ListAppstoreApp)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/appstoreapplications/:name", h.CheckByEnvironmentID, deploy.GetAppstoreApp)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/appstoreapplications", h.CheckByEnvironmentID, deploy.CreateAppstoreApp)
	rg.DELETE("/tenant/:tenant_id/project/:project_id/environment/:environment_id/appstoreapplications/:name", h.CheckByEnvironmentID, deploy.RemoveAppstoreApp)

	// 应用部署
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications", h.CheckByEnvironmentID, deploy.List)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications", h.CheckByEnvironmentID, deploy.Create)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name", h.CheckByEnvironmentID, deploy.Get)
	rg.DELETE("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name", h.CheckByEnvironmentID, deploy.Remove)
	// 应用部署镜像更新
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/_/images", h.CheckByEnvironmentID, deploy.ListImages)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/_/images", h.CheckByEnvironmentID, deploy.BatchUpdateImages)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/images", h.CheckByEnvironmentID, deploy.UpdateImages)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/images", h.CheckByEnvironmentID, deploy.GetImages)

	// 应用部署异步结果
	task := deploy.Task
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/tasks", h.CheckByEnvironmentID, task.List)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/_/tasks", h.CheckByEnvironmentID, task.BatchList)

	// 应用部署编排文件
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/files", h.CheckByEnvironmentID, deploy.ListFiles)
	rg.PUT("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/files", h.CheckByEnvironmentID, deploy.PutFiles)
	rg.PUT("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/files/:filename", h.CheckByEnvironmentID, deploy.PutFile)
	rg.DELETE("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/files/:filename", h.CheckByEnvironmentID, deploy.RemoveFile)
	// 应用部署编排文件git相关
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/gitlog", h.CheckByEnvironmentID, deploy.GitLog)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/gitdiff", h.CheckByEnvironmentID, deploy.GitDiff)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/gitrevert", h.CheckByEnvironmentID, deploy.GitRevert)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/gitpull", h.CheckByEnvironmentID, deploy.GitPull)
	// 编排内的自动补全
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/metas", h.CheckByEnvironmentID, manifest.Metas)
	// 编排作为store
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/resources/:group/:version/:kind", h.CheckByEnvironmentID, manifest.ListResource)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/resources/:group/:version/:kind/:resourcename", h.CheckByEnvironmentID, manifest.GetResource)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/resources/:group/:version/:kind/:resourcename", h.CheckByEnvironmentID, manifest.CreateResource)
	rg.PUT("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/resources/:group/:version/:kind/:resourcename", h.CheckByEnvironmentID, manifest.UpdateResource)
	rg.DELETE("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/resources/:group/:version/:kind/:resourcename", h.CheckByEnvironmentID, manifest.DeleteResource)

	// 应用部署编排更新-资源建议
	rg.PATCH("/cluster/:cluster/:group/:version/namespaces/:namespace/:resource/:name", h.CheckByClusterNamespace, deploy.UpdateWorkloadResources)
	// Argo CD相关操作
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/argohistory", h.CheckByEnvironmentID, deploy.Argohistory)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/imagehistory", h.CheckByEnvironmentID, deploy.ImageHistory)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/resourcetree", h.CheckByEnvironmentID, deploy.ResourceTree)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/argoresource", h.CheckByEnvironmentID, deploy.GetArgoResource)
	rg.DELETE("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/argoresource", h.CheckByEnvironmentID, deploy.DeleteArgoResource)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/sync", h.CheckByEnvironmentID, deploy.Sync)

	// 镜像相关
	image := ImageHandler{BaseHandler: manifest.BaseHandler}
	rg.GET("/tenant/:tenant_id/project/:project_id/images/vulnerabilities", h.CheckByProjectID, image.Vulnerabilities)
	rg.GET("/tenant/:tenant_id/project/:project_id/images/summary", h.CheckByProjectID, image.Summary)
	rg.PUT("/tenant/:tenant_id/project/:project_id/images/unpublishable", h.CheckByProjectID, image.Unpublishable)
	rg.POST("/tenant/:tenant_id/project/:project_id/images/scan", h.CheckByProjectID, image.Scan)
	rg.GET("/tenant/:tenant_id/project/:project_id/images/tags", h.CheckByProjectID, image.ImageTags)

	// 策略化发布 灰度发布
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/strategydeploy", h.CheckByEnvironmentID, deploy.GetStrategyDeployment)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/strategydeploy", h.CheckByEnvironmentID, deploy.EnableStrategyDeployment)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/strategyswitch", h.CheckByEnvironmentID, deploy.SwitchStrategy)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/analysistemplate", h.CheckByEnvironmentID, deploy.ListAnalysisTemplate)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/strategydeploystatus", h.CheckByEnvironmentID, deploy.StrategyDeploymentStatus)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/strategydeploycontrol", h.CheckByEnvironmentID, deploy.StrategyDeploymentControl)

	// 部署状态的附加信息
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/services", h.CheckByEnvironmentID, deploy.ListRelatedService)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/replicas", h.CheckByEnvironmentID, deploy.GetReplicas)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/replicas", h.CheckByEnvironmentID, deploy.SetReplicas)
	rg.GET("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/hpa", h.CheckByEnvironmentID, deploy.GetHPA)
	rg.POST("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/hpa", h.CheckByEnvironmentID, deploy.SetHPA)
	rg.DELETE("/tenant/:tenant_id/project/:project_id/environment/:environment_id/applications/:name/hpa", h.CheckByEnvironmentID, deploy.DeleteHPA)

	// ⬇️ 直接使用名称时路由全部注册为复数
	// 供外部集成使用,填充名称
	rg.POST("/tenants/:tenant/projects/:project/environments/:environment/applications/:name/images", deploy.DirectUpdateImage)
	return nil
}
