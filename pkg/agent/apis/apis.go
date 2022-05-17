package apis

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/labels"
	"kubegems.io/pkg/agent/client"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/agent/middleware"
	"kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/apis/plugins"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/prometheus/exporter"
	"kubegems.io/pkg/utils/route"
	"kubegems.io/pkg/utils/system"
	"kubegems.io/pkg/version"
)

type DebugOptions struct {
	Image       string `json:"image,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	PodSelector string `json:"podSelector,omitempty"`
	Container   string `json:"container,omitempty"`
}

func NewDefaultDebugOptions() *DebugOptions {
	return &DebugOptions{
		Namespace: os.Getenv("MY_NAMESPACE"),
		PodSelector: labels.SelectorFromSet(
			labels.Set{
				"app.kubernetes.io/name": "gems-agent-kubectl",
			}).String(),
		Container: "gems-agent-kubectl",
		Image:     "kubegems/debug-tools:latest",
	}
}

type Options struct {
	PrometheusServer   string `json:"prometheusServer,omitempty"`
	AlertmanagerServer string `json:"alertmanagerServer,omitempty"`
	LokiServer         string `json:"lokiServer,omitempty"`
	JaegerServer       string `json:"jaegerServer,omitempty"`
	EnableHTTPSigs     bool   `json:"enableHTTPSigs,omitempty" description:"check http sigs, default false"`
}

func NewDefaultOptions() *Options {
	return &Options{
		PrometheusServer:   fmt.Sprintf("http://prometheus.%s:9090", gems.NamespaceMonitor),
		AlertmanagerServer: fmt.Sprintf("http://alertmanager.%s:9093", gems.NamespaceMonitor),
		LokiServer:         fmt.Sprintf("http://loki-gateway.%s:3100", gems.NamespaceLogging),
		JaegerServer:       "http://jaeger-query.observability:16686",
		EnableHTTPSigs:     false,
	}
}

type handlerMux struct{ r *route.Router }

const (
	ActionCreate  = "create"
	ActionDelete  = "delete"
	ActionUpdate  = "update"
	ActionPatch   = "patch"
	ActionList    = "list"
	ActionGet     = "get"
	ActionCheck   = "check"
	ActionEnable  = "enable"
	ActionDisable = "disable"
)

// register
func (mu handlerMux) register(group, version, resource, action string, handler gin.HandlerFunc, method ...string) {
	switch action {
	case ActionGet:
		mu.r.MustRegister(http.MethodGet, fmt.Sprintf("/custom/%s/%s/%s/{name}", group, version, resource), handler)
		mu.r.MustRegister(http.MethodGet, fmt.Sprintf("/custom/%s/%s/namespaces/{namespace}/%s/{name}", group, version, resource), handler)
	case ActionList:
		mu.r.MustRegister(http.MethodGet, fmt.Sprintf("/custom/%s/%s/%s", group, version, resource), handler)
		mu.r.MustRegister(http.MethodGet, fmt.Sprintf("/custom/%s/%s/namespaces/{namespace}/%s", group, version, resource), handler)
	default:
		mu.r.MustRegister("*", fmt.Sprintf("/custom/%s/%s/%s/{name}/actions/%s", group, version, resource, action), handler)
		mu.r.MustRegister("*", fmt.Sprintf("/custom/%s/%s/namespaces/{namespace}/%s/{name}/actions/%s", group, version, resource, action), handler)
	}
}

// nolint: funlen
func Run(ctx context.Context, cluster cluster.Interface, system *system.Options, options *Options, debugOptions *DebugOptions) error {
	ginr := gin.New()
	ginr.Use(
		// log
		log.DefaultGinLoggerMideare(),
		// 请求数统计
		exporter.GetRequestCollector().HandlerFunc(),
		// panic recovery
		gin.Recovery(),
	)
	if options.EnableHTTPSigs {
		ginr.Use(middleware.SignerMiddleware())
	}

	rr := route.NewRouter()

	ginr.Any("/*path", func(c *gin.Context) {
		rr.Match(c)(c)
	})

	routes := handlerMux{r: rr}
	routes.r.GET("/healthz", func(c *gin.Context) {
		content, err := cluster.Kubernetes().Discovery().RESTClient().Get().AbsPath("/healthz").DoRaw(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"healthy": "notok",
				"reason":  err.Error(),
			})
			return
		}
		contentStr := string(content)
		if contentStr != "ok" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"healthy": "notok",
				"reason":  contentStr,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"healthy": "ok"})
	})
	routes.r.GET("/version", func(c *gin.Context) { c.JSON(http.StatusOK, version.Get()) })

	serviceProxyHandler := ServiceProxyHandler{}
	routes.r.ANY("/v1/service-proxy/{realpath}*", serviceProxyHandler.ServiceProxy)

	// restful api for all k8s resources
	routes.registerREST(cluster)

	// custom api
	staticsHandler := &StatisticsHandler{C: cluster.GetClient()}
	routes.register("statistics.system", "v1", "workloads", ActionList, staticsHandler.ClusterWorkloadStatistics)
	routes.register("statistics.system", "v1", "resources", ActionList, staticsHandler.ClusterResourceStatistics)

	nodeHandler := &NodeHandler{C: cluster.GetClient()}
	routes.register("core", "v1", "nodes", ActionGet, nodeHandler.Get)
	routes.register("core", "v1", "nodes", "metadata", nodeHandler.PatchNodeLabelOrAnnotations)
	routes.register("core", "v1", "nodes", "taint", nodeHandler.PatchNodeTaint)
	routes.register("core", "v1", "nodes", "cordon", nodeHandler.PatchNodeCordon)

	nsHandler := &NamespaceHandler{C: cluster.GetClient()}
	routes.register("core", "v1", "namespaces", ActionList, nsHandler.List)

	podHandler := PodHandler{cluster: cluster, debugoptions: debugOptions}
	routes.register("core", "v1", "pods", ActionList, podHandler.List)
	routes.register("core", "v1", "pods", "shell", podHandler.ExecPods)
	routes.register("core", "v1", "pods", "debug", podHandler.DebugPod)
	routes.register("core", "v1", "pods", "logs", podHandler.GetContainerLogs)

	rolloutHandler := &RolloutHandler{cluster: cluster}
	routes.register("apps", "v1", "daemonsets", "rollouthistory", rolloutHandler.DaemonSetHistory)
	routes.register("apps", "v1", "statefulsets", "rollouthistory", rolloutHandler.StatefulSetHistory)
	routes.register("apps", "v1", "deployments", "rollouthistory", rolloutHandler.DeploymentHistory)
	routes.register("apps", "v1", "daemonsets", "rollback", rolloutHandler.DaemonsetRollback)
	routes.register("apps", "v1", "statefulsets", "rollback", rolloutHandler.StatefulSetRollback)
	routes.register("apps", "v1", "deployments", "rollback", rolloutHandler.DeploymentRollback)

	kubectlHandler := KubectlHandler{cluster: cluster, debugoptions: debugOptions}
	routes.register("system", "v1", "kubectl", ActionList, kubectlHandler.ExecKubectl)

	prometheusHandler, err := NewPrometheusHandler(options.PrometheusServer)
	if err != nil {
		return err
	}
	routes.register("prometheus", "v1", "vector", ActionList, prometheusHandler.Vector)
	routes.register("prometheus", "v1", "matrix", ActionList, prometheusHandler.Matrix)
	routes.register("prometheus", "v1", "labelvalues", ActionList, prometheusHandler.LabelValues)
	routes.register("prometheus", "v1", "labelnames", ActionList, prometheusHandler.LabelNames)
	routes.register("prometheus", "v1", "alertrule", ActionList, prometheusHandler.AlertRule)
	routes.register("prometheus", "v1", "componentstatus", ActionList, prometheusHandler.ComponentStatus)
	routes.register("prometheus", "v1", "certinfos", ActionGet, prometheusHandler.CertInfo)

	alertmanagerHandler, err := NewAlertmanagerClient(options.AlertmanagerServer, cluster.Kubernetes())
	if err != nil {
		return err
	}
	routes.register("alertmanager", "v1", "alerts", ActionList, alertmanagerHandler.ListAlerts)
	routes.register("alertmanager", "v1", "alerts", ActionCheck, alertmanagerHandler.CheckConfig)
	routes.register("alertmanager", "v1", "silence", ActionList, alertmanagerHandler.ListSilence)
	routes.register("alertmanager", "v1", "silence", ActionCreate, alertmanagerHandler.CreateSilence)
	routes.register("alertmanager", "v1", "silence", ActionDelete, alertmanagerHandler.DeleteSilence)

	lokiHandler := &LokiHandler{Server: options.LokiServer}
	routes.register("loki", "v1", "query", ActionList, lokiHandler.Query)
	routes.register("loki", "v1", "queryrange", ActionList, lokiHandler.QueryRange)
	routes.register("loki", "v1", "labels", ActionList, lokiHandler.Labels)
	routes.register("loki", "v1", "labelvalues", ActionList, lokiHandler.LabelValues)
	routes.register("loki", "v1", "tail", ActionList, lokiHandler.Tail)
	routes.register("loki", "v1", "series", ActionList, lokiHandler.Series)
	routes.register("loki", "v1", "alertrule", ActionList, lokiHandler.AlertRule)

	jobHandle := &JobHandler{C: cluster.GetClient(), cluster: cluster}
	routes.register("batch", "v1", "jobs", ActionList, jobHandle.List)

	eventHandler := EventHandler{C: cluster.GetClient()}
	routes.register("core", "v1", "events", ActionList, eventHandler.List)

	pvcHandler := PvcHandler{C: cluster.GetClient()}
	routes.register("core", "v1", "pvcs", ActionList, pvcHandler.List)
	routes.register("core", "v1", "pvcs", ActionGet, pvcHandler.Get)

	secretHandler := SecretHandler{C: cluster.GetClient(), cluster: cluster}
	routes.register("core", "v1", "secrets", ActionList, secretHandler.List)

	pluginHandler := PluginHandler{cluster: cluster}
	routes.register(plugins.GroupName, "v1beta1", "installers", ActionList, pluginHandler.List)
	routes.register(plugins.GroupName, "v1beta1", "installers", ActionEnable, pluginHandler.Enable)
	routes.register(plugins.GroupName, "v1beta1", "installers", ActionDisable, pluginHandler.Disable)

	argoRolloutHandler := &ArgoRolloutHandler{cluster: cluster}
	routes.register("argoproj.io", "v1alpha1", "rollouts", "info", argoRolloutHandler.GetRolloutInfo)
	routes.register("argoproj.io", "v1alpha1", "rollouts", "depinfo", argoRolloutHandler.GetRolloutDepInfo)

	jaegerHandler := &jaegerHandler{Server: options.JaegerServer}
	routes.register("jaeger", "v1", "span", ActionList, jaegerHandler.GetSpanCount)

	// watcher 给消息中心使用的，不暴露给前端用户
	w := NewWatcher(cluster.GetCache())
	w.Start()
	routes.r.GET("/notify", w.StreamWatch)

	alertHandler := &AlertHandler{Watcher: w}
	routes.r.POST("/alert", alertHandler.Webhook)

	// service client 使用的内部 apis
	clientrest := client.ClientRest{Cli: cluster.GetClient()}
	clientrest.Register(routes.r)

	if err := listen(ctx, system, ginr); err != nil {
		return err
	}
	return nil
}

func (mu handlerMux) registerREST(cluster cluster.Interface) {
	resthandler := REST{
		client:  cluster.GetClient(),
		cluster: cluster,
	}

	mu.r.GET("/v1/{group}/{version}/{resource}", resthandler.List)
	mu.r.GET("/v1/{group}/{version}/namespaces/{namespace}/{resource}", resthandler.List)

	mu.r.GET("/v1/{group}/{version}/{resource}/{name}", resthandler.Get)
	mu.r.GET("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}", resthandler.Get)

	mu.r.POST("/v1/{group}/{version}/{resource}/{name}", resthandler.Create)
	mu.r.POST("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}", resthandler.Create)

	mu.r.PUT("/v1/{group}/{version}/{resource}/{name}", resthandler.Update)
	mu.r.PUT("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}", resthandler.Update)

	mu.r.PATCH("/v1/{group}/{version}/{resource}/{name}", resthandler.Patch)
	mu.r.PATCH("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}", resthandler.Patch)

	mu.r.DELETE("/v1/{group}/{version}/{resource}/{name}", resthandler.Delete)
	mu.r.DELETE("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}", resthandler.Delete)

	mu.r.PATCH("/v1/{group}/{version}/{resource}/{name}/actions/scale", resthandler.Scale)
	mu.r.PATCH("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}/actions/scale", resthandler.Scale)
}

func listen(ctx context.Context, options *system.Options, handler http.Handler) error {
	server := http.Server{
		BaseContext: func(l net.Listener) context.Context { return ctx },
		Addr:        options.Listen,
		Handler:     handler,
	}

	if options.IsTLSConfigEnabled() {
		tlsc, err := options.ToTLSConfig()
		if err != nil {
			return err
		}
		server.TLSConfig = tlsc
		// server.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert // enable TLS client auth
	} else {
		log.Info("tls config not found")
	}

	go func() {
		<-ctx.Done()
		log.Info("shutting down server")
		server.Close()
	}()

	if server.TLSConfig != nil {
		log.Info("listen on https", "addr", options.Listen)
		return server.ListenAndServeTLS("", "")
	} else {
		log.Info("listen on http", "addr", options.Listen)
		return server.ListenAndServe()
	}
}
