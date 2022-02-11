package application

const (
	LabelFrom              = GroupName + "/from" // 区分是从 appstore 还是从应用 app 部署的argo
	LabelValueFromApp      = "app"
	LabelValueFromAppStore = "appstore"

	AnnotationCreator   = GroupName + "/creator"   // 创建人,仅用于当前部署实时更新，从kustomize部署的历史需要从gitcommit取得
	AnnotationRef       = GroupName + "/ref"       // 标志这个资源所属的项目环境，避免使用过多label造成干扰
	AnnotationCluster   = GroupName + "/cluster"   // 标志这个资源所属集群
	AnnotationNamespace = GroupName + "/namespace" // 标志这个资源所属namespace

	AnnotationGeneratedByPlatformKey           = GroupName + "/is-generated-by" // 表示该资源是为了{for}自动生成的，需要在特定的时刻被清理
	AnnotationGeneratedByPlatformValueRollouts = "rollouts"                     // 表示该资源是为了rollouts而生成的

	AnnotationImagePullSecretKeyPrefix = GroupName + "/imagePullSecrets-"
)
