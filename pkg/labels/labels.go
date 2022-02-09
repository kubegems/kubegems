package labels

const (
	LabelTenant       = "kubegems.io/tenant"
	LabelProject      = "kubegems.io/project"
	LabelEnvironment  = "kubegems.io/environment"
	LabelApplication  = "kubegems.io/application"
	LabelZone         = "kubegems.io/zone"
	LabelPlugins      = "kubegems.io/plugins"
	LabelIngressClass = "kubegems.io/ingressClass" // ingress打标签用以筛选

	NamespaceSystem   = "gemcloud-system"
	NamespaceMonitor  = "gemcloud-monitoring-system"
	NamespaceLogging  = "gemcloud-logging-system"
	NamespaceGateway  = "gemcloud-gateway-system"
	NamespaceWorkflow = "gemcloud-workflow-system"

	// ref by outside
	ArgoLabelKeyCreator        = "kubegems.io/creator" // 创建人,仅用于当前部署实时更新，从kustomize部署的历史需要从gitcommit取得
	ArgoLabelKeyFrom           = "kubegems.io/from"    // 区分是从 appstore 还是从应用 app 部署的argo
	ArgoLabelValueFromApp      = "app"
	ArgoLabelValueFromAppStore = "appstore"

	// istio related annotations
	AnnotationVirtualDomain = "kubegems.io/virtualdomain"
	AnnotationVirtualSpace  = "kubegems.io/virtualspace"

	GemsLabelPrefix = "kubegems.io"
)

var CommonLabels = []string{
	LabelTenant,
	LabelProject,
	LabelEnvironment,
}
