package gems

const (
	LabelTenant      = GroupName + "/tenant"
	LabelProject     = GroupName + "/project"
	LabelEnvironment = GroupName + "/environment"
	LabelApplication = GroupName + "/application"
	LabelZone        = GroupName + "/zone"
	LabelPlugins     = GroupName + "/plugins"

	NamespaceSystem   = "kubegems"
	NamespaceMonitor  = "kubegems-monitoring"
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
	FinalizerNamespace     = "finalizer." + GroupName + "/namespace"
	FinalizerResourceQuota = "finalizer." + GroupName + "/resourcequota"
	FinalizerGateway       = "finalizer." + GroupName + "/gateway"
	FinalizerNetworkPolicy = "finalizer." + GroupName + "/networkpolicy"
	FinalizerLimitrange    = "finalizer." + GroupName + "/limitrange"
	FinalizerEnvironment   = "finalizer." + GroupName + "/environment"
)

const (
	LabelMonitorCollector = GroupName + "/monitoring"
	LabelLogCollector     = GroupName + "/logging"

	LabelAlertmanagerConfig = "alertmanagerconfig.kubegems.io/name"
	LabelPrometheusRule     = "prometheusrule.kubegems.io/name"

	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
)
