package plugins

const (
	AnnotationIgnoreOptions        = "bundle.kubegems.io/ignore-options"
	AnnotationIgnoreOptionOnUpdate = "OnUpdate"
	AnnotationIgnoreOptionOnDelete = "OnDelete"
)

const (
	AnnotationDescription      = "plugins.kubegems.io/description"
	AnnotationAppVersion       = "plugins.kubegems.io/appVersion"
	AnnotationCategory         = "plugins.kubegems.io/category"
	AnnotationMainCategory     = "plugins.kubegems.io/main-category"
	AnnotationIcon             = "plugins.kubegems.io/icon"
	AnnotationHealthCheck      = "plugins.kubegems.io/health-check"
	AnnotationRequired         = "plugins.kubegems.io/required"
	AnnotationIgnoreOnDisabled = "plugins.kubegems.io/ignore-on-disabled"
	AnnotationSchema           = "plugins.kubegems.io/schema"
	AnnotationUseTemplate      = "plugins.kubegems.io/use-template"
	AnnotationInstallNamespace = "plugins.kubegems.io/install-namespace"
	AnnotationValuesFrom       = "plugins.kubegems.io/values-from"
	AnnotationPluginInfo       = "plugins.kubegems.io/plugin-info"
)

const (
	KubeGemsLocalPluginsNamespace     = "kubegems-local"
	KubeGemsGlobalValuesConfigMapName = "kubegems-global-values"
)
