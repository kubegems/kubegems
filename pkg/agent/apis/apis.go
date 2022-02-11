package apis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"kubegems.io/pkg/agent/client"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/agent/middleware"
	"kubegems.io/pkg/apis/plugins"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/prometheus/collector"
	"kubegems.io/pkg/utils/route"
	"kubegems.io/pkg/version"
)

type DebugOptions struct {
	DebugToolsImage string

	MyNamespace string
	MyPod       string
	MyContainer string
}

func (o *DebugOptions) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&o.DebugToolsImage, utils.JoinFlagName(prefix, "debugtoolsimage"), o.DebugToolsImage, "debug tools image")
	fs.StringVar(&o.MyContainer, utils.JoinFlagName(prefix, "mycontainer"), o.MyContainer, "self container name")
	fs.StringVar(&o.MyNamespace, utils.JoinFlagName(prefix, "mynamespace"), o.MyNamespace, "self pod namespace")
	fs.StringVar(&o.MyPod, utils.JoinFlagName(prefix, "mypod"), o.MyPod, "self pod name")
}

type Options struct {
	ListenTLS string
	Cert      string
	Key       string
	CA        string
	CertDir   string

	Listen string

	MetricsServer      string
	PrometheusServer   string
	AlertmanagerServer string
	LokiServer         string
	JaegerSerber       string
	EnableHTTPSigs     bool

	DebugOptions *DebugOptions
}

func (o *Options) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVarP(&o.Listen, utils.JoinFlagName(prefix, "listen"), "l", o.Listen, "listen addr")
	fs.StringVarP(&o.ListenTLS, utils.JoinFlagName(prefix, "listentls"), "", o.ListenTLS, "listen tls addr")
	fs.StringVar(&o.CertDir, utils.JoinFlagName(prefix, "certdir"), o.CertDir, "cert files dir")
	fs.StringVar(&o.CA, utils.JoinFlagName(prefix, "ca"), o.CA, "ca bundles filename")
	fs.StringVar(&o.Cert, utils.JoinFlagName(prefix, "cert"), o.Cert, "listen cert filename")
	fs.StringVar(&o.Key, utils.JoinFlagName(prefix, "key"), o.Key, "listen key filename")

	fs.StringVar(&o.MetricsServer, utils.JoinFlagName(prefix, "metricsserver"), o.MetricsServer, "metrics server addr")
	fs.StringVar(&o.PrometheusServer, utils.JoinFlagName(prefix, "prometheusserver"), o.PrometheusServer, "prometheus server addr")
	fs.StringVar(&o.LokiServer, utils.JoinFlagName(prefix, "lokiserver"), o.LokiServer, "loki server addr")
	fs.StringVar(&o.AlertmanagerServer, utils.JoinFlagName(prefix, "alertmanagerserver"), o.AlertmanagerServer, "alertmanager server addr")
	fs.StringVar(&o.JaegerSerber, utils.JoinFlagName(prefix, "jaegerserver"), o.JaegerSerber, "jaeger server addr")
	fs.BoolVar(&o.EnableHTTPSigs, utils.JoinFlagName(prefix, "enablehttpsigs"), o.EnableHTTPSigs, "enable http sigs")

	o.DebugOptions.RegistFlags("debugoptions", fs)
}

func DefaultOptions() *Options {
	debugPodname, _ := os.LookupEnv("MY_POD_NAME")
	debugNamespace, _ := os.LookupEnv("MY_NAMESPACE")
	return &Options{
		Listen:             ":8041",
		ListenTLS:          ":8040",
		MetricsServer:      "http://metrics-scraper.gemcloud-monitoring-system:8000",
		PrometheusServer:   "http://prometheus.gemcloud-monitoring-system:9090",
		AlertmanagerServer: "http://alertmanager.gemcloud-monitoring-system:9093",
		LokiServer:         "http://loki-gateway.gemcloud-logging-system:3100",
		JaegerSerber:       "http://jaeger-query.observability:16686",
		EnableHTTPSigs:     true,
		DebugOptions: &DebugOptions{
			MyNamespace:     debugNamespace,
			MyPod:           debugPodname,
			MyContainer:     "gems-agent-kubectl",
			DebugToolsImage: "kubegems/debug-tools:latest",
		},
		CertDir: "certs",
		Cert:    "tls.crt",
		Key:     "tls.key",
		CA:      "ca.crt",
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
func Run(ctx context.Context, cluster cluster.Interface, options *Options) error {
	ginr := gin.New()
	ginr.Use(
		// log
		log.DefaultGinLoggerMideare(),
		// 请求数统计
		collector.GetRequestCollector().HandlerFunc(),
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

	metricsHandler := &MetricsHandler{metricScraperHost: options.MetricsServer, C: cluster.GetClient()}
	routes.register("metrics.k8s.io", "v1beta1", "nodes", ActionList, metricsHandler.Nodes)
	routes.register("metrics.k8s.io", "v1beta1", "nodes", ActionGet, metricsHandler.Node)
	routes.register("metrics.k8s.io", "v1beta1", "nodes", "recently", metricsHandler.NodeList)
	routes.register("metrics.k8s.io", "v1beta1", "pods", ActionList, metricsHandler.Pods)
	routes.register("metrics.k8s.io", "v1beta1", "pods", "recently", metricsHandler.PodList)

	podHandler := PodHandler{cluster: cluster, debugoptions: options.DebugOptions}
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

	kubectlHandler := KubectlHandler{cluster: cluster, debugoptions: options.DebugOptions}
	routes.register("system", "v1", "kubectl", ActionList, kubectlHandler.ExecKubectl)

	prometheusHandler, err := NewPrometheusHandler(options.PrometheusServer, cluster)
	if err != nil {
		return err
	}
	routes.register("prometheus", "v1", "vector", ActionList, prometheusHandler.Vector)
	routes.register("prometheus", "v1", "matrix", ActionList, prometheusHandler.Matrix)
	routes.register("prometheus", "v1", "labelvalues", ActionList, prometheusHandler.LabelValues)
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
	routes.register("loki", "v1", "queryrange", ActionList, lokiHandler.QueryRange)
	routes.register("loki", "v1", "labels", ActionList, lokiHandler.Labels)
	routes.register("loki", "v1", "labelvalues", ActionList, lokiHandler.LabelValues)
	routes.register("loki", "v1", "tail", ActionList, lokiHandler.Tail)
	routes.register("loki", "v1", "series", ActionList, lokiHandler.Series)

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
	routes.register(plugins.GroupName, "v1alpha1", "plugins", ActionList, pluginHandler.List)
	routes.register(plugins.GroupName, "v1alpha1", "plugins", ActionEnable, pluginHandler.Enable)
	routes.register(plugins.GroupName, "v1alpha1", "plugins", ActionDisable, pluginHandler.Disable)

	argoRolloutHandler := &ArgoRolloutHandler{cluster: cluster}
	routes.register("argoproj.io", "v1alpha1", "rollouts", "info", argoRolloutHandler.GetRolloutInfo)
	routes.register("argoproj.io", "v1alpha1", "rollouts", "depinfo", argoRolloutHandler.GetRolloutDepInfo)

	jaegerHandler := &jaegerHandler{Server: options.JaegerSerber}
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

	if err := listen(ctx, options, ginr); err != nil {
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

func listen(ctx context.Context, options *Options, handler http.Handler) error {
	basicserver := http.Server{
		BaseContext: func(l net.Listener) context.Context { return ctx },
		Addr:        options.Listen,
		Handler:     handler,
	}
	tlsserver := http.Server{
		BaseContext: func(l net.Listener) context.Context { return ctx },
		Addr:        options.ListenTLS,
		Handler:     handler,
	}

	errch := make(chan error)
	go func() {
		log.WithField("listen", options.Listen).Info("http listening")
		errch <- basicserver.ListenAndServe()
	}()
	go func() {
		if options.CA != "" && options.Cert != "" && options.Key != "" {
			ca, cert, key := filepath.Join(options.CertDir, options.CA), filepath.Join(options.CertDir, options.Cert), filepath.Join(options.CertDir, options.Key)
			// setup tls config and client certificate verify
			tlsconfig, err := TLSConfigFrom(ca, cert, key)
			if err != nil {
				errch <- err
				return
			}
			tlsserver.TLSConfig = tlsconfig
			log.WithField("listen", options.ListenTLS).Info("https listening")
			errch <- tlsserver.ListenAndServeTLS(cert, key)
		} else {
			log.Warnf("https listen not started due to no tls config")
		}
	}()

	select {
	case err := <-errch:
		return err
	case <-ctx.Done():
		basicserver.Close()
		tlsserver.Close()
		return nil
	}
}

func TLSConfigFrom(ca, cert, key string) (*tls.Config, error) {
	capem, err := ioutil.ReadFile(ca)
	if err != nil {
		return nil, err
	}
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	certPool.AppendCertsFromPEM(capem)
	certificate, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		ClientCAs:    certPool,
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}
