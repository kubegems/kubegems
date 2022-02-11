package gems

const (
	LabelTenant       = "gems.kubegems.io/tenant"
	LabelProject      = "gems.kubegems.io/project"
	LabelEnvironment  = "gems.kubegems.io/environment"
	LabelApplication  = "gems.kubegems.io/application"
	LabelZone         = "gems.kubegems.io/zone"
	LabelPlugins      = "gems.kubegems.io/plugins"
	LabelIngressClass = "gems.kubegems.io/ingressClass" // ingress打标签用以筛选

	NamespaceSystem   = "gemcloud-system"
	NamespaceMonitor  = "gemcloud-monitoring-system"
	NamespaceLogging  = "gemcloud-logging-system"
	NamespaceGateway  = "gemcloud-gateway-system"
	NamespaceWorkflow = "gemcloud-workflow-system"
)

var CommonLabels = []string{
	LabelTenant,
	LabelProject,
	LabelEnvironment,
}

const (
	FinalizerNamespace     = "finalizer.gems.kubegems.io/namespace"
	FinalizerResourceQuota = "finalizer.gems.kubegems.io/resourcequota"
	FinalizerGateway       = "finalizer.gems.kubegems.io/gateway"
	FinalizerNetworkPolicy = "finalizer.gems.kubegems.io/networkpolicy"
	FinalizerLimitrange    = "finalizer.gems.kubegems.io/limitrange"
	FinalizerEnvironment   = "finalizer.gems.kubegems.io/environment"
)
