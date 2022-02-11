package application

const (
	LabelFrom              = "application.kubegems.io/from" // 区分是从 appstore 还是从应用 app 部署的argo
	LabelValueFromApp      = "app"
	LabelValueFromAppStore = "appstore"

	AnnotationCreator   = "application.kubegems.io/creator"   // 创建人,仅用于当前部署实时更新，从kustomize部署的历史需要从gitcommit取得
	AnnotationRef       = "application.kubegems.io/ref"       // 标志这个资源所属的项目环境，避免使用过多label造成干扰
	AnnotationCluster   = "application.kubegems.io/cluster"   // 标志这个资源所属集群
	AnnotationNamespace = "application.kubegems.io/namespace" // 标志这个资源所属namespace
)
