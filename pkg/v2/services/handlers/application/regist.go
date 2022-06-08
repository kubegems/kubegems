package application

import (
	"github.com/argoproj/argo-rollouts/pkg/apiclient/rollout"
	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/emicklei/go-restful/v3"
	"github.com/goharbor/harbor/src/pkg/scan/vuln"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/route"
	"kubegems.io/kubegems/pkg/utils/workflow"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
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

func (h *ApplicationHandler) Regist(c *restful.Container) {
	ws := &restful.WebService{}
	h.Register(ws)
	c.Add(ws)
}

// nolint: funlen,maintidex
func (h *ApplicationHandler) Register(ws *restful.WebService) {
	deploy := h
	manifest := h.Manifest
	task := deploy.Task
	image := &ImageHandler{BaseHandler: manifest.BaseHandler}

	newfilesGroup := func() *route.Group {
		return route.
			NewGroup("").
			Tag("resources").
			AddRoutes(
				// git
				route.GET("/gitlog").To(manifest.GitLog).
					Paged().
					Response(GitLog{}),
				route.GET("/gitdiff").To(manifest.GitDiff).
					Parameters(route.QueryParameter("hash", "git hash")).
					Response([]git.FileDiff{}),
				route.POST("/gitrevert").To(manifest.GitRevert).
					Parameters(route.QueryParameter("hash", "git hash")),
				route.POST("/gitpull").To(manifest.GitPull),
				// meta
				route.GET("/metas").To(manifest.ListMetas).
					Parameters(route.QueryParameter("kind", "filte kind").Optional()),
			).
			AddSubGroup(
				route.NewGroup("/files").
					AddRoutes(
						route.GET("").To(manifest.ListFiles).Response([]FileContent{}, "files"),
						route.POST("").To(manifest.UpdateFiles).
							Parameters(
								route.BodyParameter("files", []FileContent{}),
								route.QueryParameter("msg", "commit message"),
							),
						route.GET("/{file}").To(manifest.GetFile).Response(FileContent{}), // not implemented
						route.PUT("/{file}").To(manifest.UpdateFile).
							Parameters(
								route.PathParameter("file", "file name"),
								route.BodyParameter("content", []FileContent{}),
							),
						route.DELETE("/{file}").To(manifest.DeleteFile).Parameters(
							route.PathParameter("file", "file name"),
						),
					),
				route.NewGroup("/resources/{group}/{version}/{kind}").
					Parameters(
						route.PathParameter("group", "group name"),
						route.PathParameter("version", "version name"),
						route.PathParameter("kind", "kind name"),
					).
					AddRoutes(
						route.GET("").To(manifest.ListResources).Response(unstructured.UnstructuredList{}),
						route.POST("").To(manifest.CreateResource).Parameters(
							route.BodyParameter("resource", unstructured.Unstructured{}),
						),
						route.GET("{name}").To(manifest.GetResource).
							Parameters(route.PathParameter("name", "resource name")).
							Response(unstructured.Unstructured{}),
						route.PUT("{name}").To(manifest.UpdateResource).
							Parameters(
								route.PathParameter("name", "resource name"),
								route.BodyParameter("resource", unstructured.Unstructured{}),
							),
						route.DELETE("{name}").To(manifest.DeleteResource).
							Parameters(route.PathParameter("name", "resource name")),
					),
			)
	}

	tree := &route.Tree{
		RouteUpdateFunc: func(r *route.Route) {
			page, size := false, false
			for _, param := range r.Params {
				if param.Kind == route.ParamKindQuery {
					if param.Name == "page" {
						page = true
					}
					if param.Name == "size" {
						size = true
					}
				}
			}
			for i, v := range r.Responses {
				//  if query parameters exist, response as a paged response
				if page && size {
					r.Responses[i].Body = handlers.Response{
						Data: handlers.PageData{
							List: v.Body,
						},
					}
				} else {
					r.Responses[i].Body = handlers.Response{
						Data: v.Body,
					}
				}
			}
		},
		Group: route.
			NewGroup("/v2").
			Tag("reources").
			AddRoutes(
				// resource suggestion
				route.
					PATCH("/clusters/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name}").
					To(deploy.UpdateWorkloadResources).
					Parameters(
						route.PathParameter("cluster", "cluster name"),
						route.PathParameter("group", "group name"),
						route.PathParameter("version", "version name"),
						route.PathParameter("namespace", "namespace name"),
						route.PathParameter("resource", "resource name"),
						route.PathParameter("name", "resource name"),
					),
			).
			AddSubGroup(
				route.
					NewGroup("/tenants/{tenant}/projects/{project}").
					Parameters(
						route.PathParameter("tenant", "tenant name"),
						route.PathParameter("project", "project name"),
					).
					AddSubGroup(
						route.
							NewGroup("/images").
							Tag("project-images").
							Parameters(route.QueryParameter("image", "image name")).
							AddRoutes(
								route.GET("/vulnerabilities").To(image.Vulnerabilities).
									Response(vuln.Report{}),
								route.GET("/summary").To(image.Summary).
									Paged().
									Response([]ImageSummaryItem{}, "paged image summary"),
								route.POST("/unpublishable").To(image.Unpublishable).
									Parameters(
										route.QueryParameter("unpublishable", "true/false").DataType("boolean"),
									),
								route.POST("/scan").To(image.Scan),
								route.GET("/tags").To(image.ImageTags).
									Response([]ImageTag{}),
							),
						route.
							NewGroup("/manifests").
							Tag("manifest").
							AddRoutes(
								route.GET("").To(manifest.ListManifest).
									Paged().
									Response([]Manifest{}, "paged manifests"),
								route.POST("").To(manifest.CreateManifest).
									Parameters(route.BodyParameter("manifest", Manifest{})),
							).
							AddSubGroup(
								route.
									NewGroup("/{manifest}").
									Tag("manifest").
									Parameters(route.PathParameter("manifest", "manifest name")).
									AddRoutes(
										route.GET("").To(manifest.GetManifest).
											Response(Manifest{}, "manifest"),
										route.PUT("").To(manifest.UpdateManifest).
											Parameters(route.BodyParameter("manifest", Manifest{})),
										route.DELETE("").To(manifest.DeleteManifest),
										route.GET("/argohistory").To(deploy.ArgoHistory).
											Paged().
											Response(ArgoHistory{}, "argo history"),
										route.GET("/imagehistory").To(deploy.ImageHistory).
											Paged().
											Response(ImageHistory{}, "paged image history"),
									).
									AddSubGroup(
										newfilesGroup().
											Tag("manifest files"),
									),
							),
						route.
							NewGroup("/environments/{environment}/applications").
							Tag("application").
							Parameters(route.PathParameter("environment", "environment name")).
							AddRoutes(
								route.GET("").To(deploy.List).
									Response([]DeploiedManifest{}, "paged deployed manifests"),
								route.POST("").To(deploy.Create).
									Parameters(route.BodyParameter("deploied", DeploiedManifest{})),
								route.GET("/_/images").To(deploy.ListImages).
									Response([]DeployImages{}),
								route.POST("/_/images").To(deploy.BatchUpdateImages).
									Parameters(route.BodyParameter("images", []DeployImages{})),
								route.GET("/_/tasks").To(task.BatchList).
									Parameters(
										route.QueryParameter("names", "filter names splited by comma").Optional(),
									).
									Response([]workflow.Task{}),
							).
							AddSubGroup(
								route.
									NewGroup("/{application}").
									Tag("application").
									Parameters(
										route.PathParameter("application", "application name"),
									).
									AddRoutes(
										route.GET("").To(deploy.Get).
											Response(DeploiedManifest{}),
										route.DELETE("").To(deploy.Delete),
										// image
										route.POST("/image").To(deploy.DirectUpdateImage).
											Parameters(
												route.QueryParameter("image", "image name"),
												route.QueryParameter("version", "istio version").Optional(),
											),
										route.GET("/images").To(deploy.GetImages).
											Response(DeployImages{}),
										route.POST("/images").To(deploy.UpdateImages).
											Parameters(route.BodyParameter("images", DeployImages{})),
										// task
										route.GET("/tasks").To(task.List).
											Parameters(
												route.QueryParameter("watch", "start ssevent").Optional(),
												route.QueryParameter("type", "filter event type").Optional(),
												route.QueryParameter("limit", "limit return results").Optional(),
											).
											Response([]workflow.Task{}),
										// argo
										route.GET("/argoresource").To(deploy.GetArgoResource).
											Parameters(
												route.QueryParameter("namespace", "namespace"),
												route.QueryParameter("name", "name"),
												route.QueryParameter("group", "group"),
												route.QueryParameter("kind", "kind"),
												route.QueryParameter("version", "version"),
											).
											Response(ArgoResourceDiff{}),
										route.DELETE("/argoresource").To(deploy.DeleteArgoResource).
											Parameters(
												route.QueryParameter("namespace", "namespace"),
												route.QueryParameter("name", "name"),
												route.QueryParameter("group", "group"),
												route.QueryParameter("kind", "kind"),
												route.QueryParameter("version", "version"),
											),
										route.GET("/resourcetree").To(deploy.ResourceTree),
										route.POST("/sync").To(deploy.Sync).
											Parameters(
												route.BodyParameter("sync", SyncRequest{}).Optional(),
											),
										route.GET("/argohistory").To(deploy.ArgoHistory).
											Paged().
											Response([]*ArgoHistory{}, "paged argo history"),
										route.GET("/imagehistory").To(deploy.ImageHistory).
											Paged().
											Response([]*ImageHistory{}, "paged image history"),
									).
									AddSubGroup(
										// application strategydeploy
										route.
											NewGroup("").
											Tag("strategydeploy").
											AddRoutes(
												route.GET("/strategydeploy").To(deploy.GetStrategyDeployment).
													Response(DeploymentStrategyWithImages{}),
												route.POST("/strategydeploy").To(deploy.EnableStrategyDeployment).
													Parameters(
														route.BodyParameter("strategydeploy", DeploymentStrategyWithImages{}),
													),
												route.POST("/strategyswitch").To(deploy.SwitchStrategy).
													Parameters(
														route.BodyParameter("strategydeploy", DeploymentStrategy{}),
													),
												route.GET("/analysistemplate").To(deploy.ListAnalysisTemplate).
													Response([]rolloutsv1alpha1.ClusterAnalysisTemplate{}),
												route.GET("/strategydeploystatus").To(deploy.StrategyDeploymentStatus).
													Response(rollout.RolloutInfo{}),
												route.POST("/strategydeploycontrol").To(deploy.StrategyDeploymentControl).
													Parameters(
														route.BodyParameter("control", StrategyDeploymentControl{}),
													),
											),
										route.
											NewGroup("").
											Tag("application additional").
											AddRoutes(
												// application additional
												route.GET("/services").To(deploy.ListRelatedService).
													Response([]RelatedService{}),
												// application replicas
												route.GET("/replicas").To(deploy.GetReplicas).
													Response(AppReplicas{}),
												route.POST("/replicas").To(deploy.SetReplicas).
													Parameters(
														route.BodyParameter("replicas", AppReplicas{}),
													),
												// application hpa
												route.GET("/hpa").To(deploy.GetHPA).
													Response(HPAMetrics{}),
												route.POST("/hpa").To(deploy.SetHPA).
													Parameters(
														route.BodyParameter("hpa", HPAMetrics{}),
													),
												route.DELETE("/hpa").To(deploy.DeleteHPA),
											),
										newfilesGroup().
											Tag("application files"),
									),
							),
						route.
							NewGroup("/appstoreapplications").
							Tag("appstore").
							AddRoutes(
								route.GET("").To(deploy.ListAppstoreApp).
									Paged().
									Parameters(route.QueryParameter("search", "search name").Optional()).
									Response([]DeploiedHelm{}),
								route.POST("").To(deploy.CreateAppstoreApp).
									Parameters(route.BodyParameter("helm values", AppStoreDeployForm{})),
								route.GET("/{application}").To(deploy.GetAppstoreApp).
									Parameters(route.PathParameter("application", "application name")).
									Response(DeploiedManifest{}),
								route.DELETE("/{application}").To(deploy.DeleteAppstoreApp).
									Parameters(route.PathParameter("application", "application name")),
							),
					),
			),
	}
	tree.AddToWebService(ws)
}
