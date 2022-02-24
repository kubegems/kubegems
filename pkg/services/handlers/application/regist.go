package application

import (
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/argo"
)

type ApplicationHandler struct {
	BaseHandler
	ApplicationProcessor *ApplicationProcessor
	Agents               *agents.ClientSet
	Manifest             ManifestHandler
	Task                 *TaskHandler
	// self cluster cli
	ArgoCD *argo.Client
}

const (
	applicationTag = "application"
)

// nolint: lll,funlen
func (h *ApplicationHandler) Regist(container *restful.Container) {
	deploy := h
	manifest := h.Manifest
	task := deploy.Task
	image := ImageHandler{BaseHandler: manifest.BaseHandler}

	ws := new(restful.WebService)
	ws.Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	routers := []*restful.RouteBuilder{
		// manifests
		ws.GET("/tenants/{tenant}/projects/{project}/manifests").To(manifest.ListManifest),
		ws.POST("/tenants/{tenant}/projects/{project}/manifests").To(manifest.CreateManifest),
		ws.GET("/tenants/{tenant}/projects/{project}/manifests/{manifest}").To(manifest.GetManifest),
		ws.PUT("/tenants/{tenant}/projects/{project}/manifests/{manifest}").To(manifest.UpdateManifest),
		ws.DELETE("/tenants/{tenant}/projects/{project}/manifests/{manifest}").To(manifest.DeleteManifest),

		// manifest files
		ws.GET("/tenants/{tenant}/projects/{project}/manifests/{manifest}/files").To(manifest.ListFiles),
		ws.GET("/tenants/{tenant}/projects/{project}/manifests/{manifest}/files/{file}").To(manifest.GetFile),
		ws.PUT("/tenants/{tenant}/projects/{project}/manifests/{manifest}/files/{file}").To(manifest.UpdateFile),
		ws.PUT("/tenants/{tenant}/projects/{project}/manifests/{manifest}/files").To(manifest.UpdateFiles),
		ws.DELETE("/tenants/{tenant}/projects/{project}/manifests/{manifest}/files/{file}").To(manifest.DeleteFile),

		// manifest git
		ws.GET("/tenants/{tenant}/projects/{project}/manifests/{manifest}/gitlog").To(manifest.GitLog),
		ws.GET("/tenants/{tenant}/projects/{project}/manifests/{manifest}/gitdiff").To(manifest.GitDiff),
		ws.GET("/tenants/{tenant}/projects/{project}/manifests/{manifest}/gitrevert").To(manifest.GitRevert),
		ws.POST("/tenants/{tenant}/projects/{project}/manifests/{manifest}/gitpull").To(manifest.GitPull),
		// metas
		ws.GET("/tenants/{tenant}/projects/{project}/manifests/{manifest}/metas").To(manifest.ListMetas),

		// manifest resources
		ws.GET("/tenant/{tenant}/projects/{project}/manifests/{manifest}/resources/{group}/{version}/{kind}").To(manifest.ListResources),
		ws.GET("/tenant/{tenant}/projects/{project}/manifests/{manifest}/resources/{group}/{version}/{kind}/{name}").To(manifest.GetResource),
		ws.POST("/tenant/{tenant}/projects/{project}/manifests/{manifest}/resources/{group}/{version}/{kind}").To(manifest.CreateResource),
		ws.PUT("/tenant/{tenant}/projects/{project}/manifests/{manifest}/resources/{group}/{version}/{kind}/{name}").To(manifest.UpdateResource),
		ws.DELETE("/tenant/{tenant}/projects/{project}/manifests/{manifest}/resources/{group}/{version}/{kind}/{name}").To(manifest.DeleteResource),

		// argo
		ws.GET("/tenants/{tenant}/projects/{project}/manifests/{manifest}/argohistory").To(deploy.ArgoHistory),
		ws.GET("/tenants/{tenant}/projects/{project}/manifests/{manifest}/imagehistory").To(deploy.ImageHistory),

		// appstore
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/appstoreapplications").To(deploy.ListAppstoreApp),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/appstoreapplications/{app}").To(deploy.GetAppstoreApp),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/appstoreapplications").To(deploy.CreateAppstoreApp),
		ws.DELETE("/tenants/{tenant}/projects/{project}/environments/{environment}/appstoreapplications/{app}").To(deploy.DeleteAppstoreApp),

		// application
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications").To(deploy.List),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications").To(deploy.Create),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}").To(deploy.Get),
		ws.DELETE("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}").To(deploy.Delete),

		// application images
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/_/images").To(deploy.ListImages),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/_/images").To(deploy.BatchUpdateImages),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/images").To(deploy.GetImages),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/images").To(deploy.UpdateImages),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/image").To(deploy.DirectUpdateImage),

		// application tasks
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/tasks").To(task.List),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/_/tasks").To(task.BatchList),

		// application files
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/files").To(deploy.ListFiles),
		ws.PUT("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/files/{file}").To(deploy.UpdateFile),
		ws.PUT("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/files/{file}").To(deploy.UpdateFiles),
		ws.DELETE("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/files/{file}").To(deploy.DeleteFile),

		// application git
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/gitlog").To(deploy.GitLog),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/gitdiff").To(deploy.GitDiff),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/gitrevert").To(deploy.GitRevert),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/gitpull").To(deploy.GitPull),

		// application metas
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/metas").To(manifest.ListMetas),
		// application resources
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/resources/{group}/{version}/{kind}").To(manifest.ListResources),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/resources/{group}/{version}/{kind}/{name}").To(manifest.GetResource),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/resources/{group}/{version}/{kind}").To(manifest.CreateResource),
		ws.PUT("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/resources/{group}/{version}/{kind}/{name}").To(manifest.UpdateResource),
		ws.DELETE("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/resources/{group}/{version}/{kind}/{name}").To(manifest.DeleteResource),

		// resource suggestion
		ws.PATCH("/clusters/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name}").To(deploy.UpdateWorkloadResources),

		// application argo
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/argohistory").To(deploy.ArgoHistory),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/imagehistory").To(deploy.ImageHistory),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/resourcetree").To(deploy.ResourceTree),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/argoresource").To(deploy.GetArgoResource),
		ws.DELETE("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/argoresource").To(deploy.DeleteArgoResource),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/sync").To(deploy.Sync),

		// images
		ws.GET("/tenants/{tenant}/projects/{project}/images/vulnerabilities").To(image.Vulnerabilities),
		ws.GET("/tenants/{tenant}/projects/{project}/images/summary").To(image.Summary),
		ws.GET("/tenants/{tenant}/projects/{project}/images/unpublishable").To(image.Unpublishable),
		ws.GET("/tenants/{tenant}/projects/{project}/images/scan").To(image.Scan),
		ws.GET("/tenants/{tenant}/projects/{project}/images/tags").To(image.ImageTags),

		// application strategydeploy
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/strategydeploy").To(deploy.GetStrategyDeployment),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/strategydeploy").To(deploy.EnableStrategyDeployment),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/strategyswitch").To(deploy.SwitchStrategy),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/analysistemplate").To(deploy.ListAnalysisTemplate),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/strategydeploystatus").To(deploy.StrategyDeploymentStatus),
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/strategydeploycontrol").To(deploy.StrategyDeploymentControl),

		// application addtional
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/sevices").To(deploy.ListRelatedService),
		// application replicas
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/replicas").To(deploy.GetReplicas),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/replicas").To(deploy.SetReplicas),
		// application hpa
		ws.GET("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/hpa").To(deploy.GetHPA),
		ws.POST("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/hpa").To(deploy.SetHPA),
		ws.DELETE("/tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/hpa").To(deploy.DeleteHPA),
	}

	for _, route := range routers {
		ws.Route(
			route.Metadata(restfulspec.KeyOpenAPITags, applicationTag),
		)
	}
}
