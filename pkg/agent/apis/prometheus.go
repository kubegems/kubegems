package apis

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils/clusterinfo"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

func dynamicTimeStep(start time.Time, end time.Time) time.Duration {
	interval := end.Sub(start)
	if interval < 30*time.Minute {
		return 30 * time.Second // 30 分钟以内，step为30s, 返回60个点以内
	} else {
		return interval / 60 // 返回60个点，动态step
	}
}

func NewPrometheusHandler(server string) (*prometheusHandler, error) {
	client, err := api.NewClient(api.Config{Address: server})
	if err != nil {
		return nil, err
	}
	return &prometheusHandler{client: client}, nil
}

type prometheusHandler struct {
	client api.Client
}

// https://prometheus.io/docs/prometheus/latest/querying/operators/#comparison-binary-operators
var stateMap = map[string]int{
	"inactive": 1,
	"pending":  2,
	"firing":   3,
}

// @Tags         Agent.V1
// @Summary      Prometheus Vector
// @Description  Prometheus Vector
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                true   "cluster"
// @Param        query    query     string                                false  "query"
// @Param        notnull  query     bool                                  false  "notnull"
// @Success      200      {object}  handlers.ResponseStruct{Data=object}  "vector"
// @Router       /v1/proxy/cluster/{cluster}/custom/prometheus/v1/vector [get]
// @Security     JWT
func (p *prometheusHandler) Vector(c *gin.Context) {
	query := c.Query("query")

	v1api := v1.NewAPI(p.client)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	query, _ = url.QueryUnescape(query)
	obj, _, err := v1api.Query(ctx, query, time.Now())
	if err != nil {
		NotOK(c, err)
		return
	}
	if notnull, _ := strconv.ParseBool(c.Query("notnull")); notnull {
		if obj.String() == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, handlers.ResponseStruct{
				Message:   "空查询",
				Data:      nil,
				ErrorData: "空查询",
			})
			return
		}
	}
	OK(c, obj)
}

// @Tags         Agent.V1
// @Summary      Prometheus Matrix
// @Description  Prometheus Matrix
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                true   "cluster"
// @Param        start    query     string                                false  "start"
// @Param        end      query     string                                false  "end"
// @Param        step     query     int                                   false  "step, 单位秒"
// @Param        query    query     string                                false  "query"
// @Success      200      {object}  handlers.ResponseStruct{Data=object}  "matrix"
// @Router       /v1/proxy/cluster/{cluster}/custom/prometheus/v1/matrix [get]
// @Security     JWT
func (p *prometheusHandler) Matrix(c *gin.Context) {
	query := c.Query("query")
	start := c.Query("start")
	end := c.Query("end")
	step, _ := strconv.Atoi(c.Query("step"))

	v1api := v1.NewAPI(p.client)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	s, _ := time.Parse("2006-01-02T15:04:05Z", start)
	e, _ := time.Parse("2006-01-02T15:04:05Z", end)
	r := v1.Range{
		Start: s,
		End:   e,
	}

	if step > 0 {
		r.Step = time.Duration(step) * time.Second
	} else {
		// 不传step就动态控制
		r.Step = dynamicTimeStep(r.Start, r.End)
	}

	query, _ = url.QueryUnescape(query)
	obj, _, err := v1api.QueryRange(ctx, query, r)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, obj)
}

// @Tags         Agent.V1
// @Summary      Prometheus Labelnames
// @Description  Prometheus Labelnames
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                true   "cluster"
// @Param        start    query     string                                false  "start"
// @Param        end      query     string                                false  "end"
// @Param        match    query     string                                false  "query"
// @Success      200      {object}  handlers.ResponseStruct{Data=object}  "labels"
// @Router       /v1/proxy/cluster/{cluster}/custom/prometheus/v1/labelnames [get]
// @Security     JWT
func (p *prometheusHandler) LabelNames(c *gin.Context) {
	match := c.QueryArray("match")
	s, errs := time.Parse("2006-01-02T15:04:05Z", c.Query("start"))
	e, erre := time.Parse("2006-01-02T15:04:05Z", c.Query("end"))
	if errs != nil || erre != nil {
		s = time.Now().AddDate(-1, 0, 0)
		e = time.Now().AddDate(1, 0, 0)
	}

	v1api := v1.NewAPI(p.client)
	labels, warns, err := v1api.LabelNames(context.Background(), match, s, e)
	if err != nil {
		NotOK(c, err)
		return
	}
	OK(c, map[string]interface{}{
		"labels": labels,
		"warns":  warns,
	})
}

// @Tags         Agent.V1
// @Summary      Prometheus LabelValues
// @Description  Prometheus LabelValues
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                true   "cluster"
// @Param        start    query     string                                false  "start"
// @Param        end      query     string                                false  "end"
// @Param        match    query     string                                false  "query"
// @Param        label    query     string                                false  "label"
// @Param        search   query     string                                false  "search"
// @Success      200      {object}  handlers.ResponseStruct{Data=object}  "labelvalues"
// @Router       /v1/proxy/cluster/{cluster}/custom/prometheus/v1/labelvalues [get]
// @Security     JWT
func (p *prometheusHandler) LabelValues(c *gin.Context) {
	label := c.DefaultQuery("label", "__name__")
	match := c.QueryArray("match")
	s, errs := time.Parse("2006-01-02T15:04:05Z", c.Query("start"))
	e, erre := time.Parse("2006-01-02T15:04:05Z", c.Query("end"))
	if errs != nil || erre != nil {
		s = time.Now().AddDate(-1, 0, 0)
		e = time.Now().AddDate(1, 0, 0)
	}

	v1api := v1.NewAPI(p.client)
	alllabels, warns, err := v1api.LabelValues(context.Background(), label, match, s, e)
	if err != nil {
		NotOK(c, err)
		return
	}

	// 避免append
	tmp := make([]model.LabelValue, len(alllabels))
	index := -1
	for _, v := range alllabels {
		if strings.Contains(string(v), c.Query("search")) {
			index++
			tmp[index] = v
		}
	}

	// 限制最多100条
	var ret []model.LabelValue
	if index > 99 {
		ret = tmp[:100]
	} else {
		ret = tmp[:index+1]
	}

	OK(c, map[string]interface{}{
		"labels": ret,
		"warns":  warns,
	})
}

// @Tags         Agent.V1
// @Summary      Prometheus alertrule
// @Description  Prometheus 获取告警规则详情
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                                                 true  "cluster"
// @Param        name     query     string                                                                 true  "name"
// @Success      200      {object}  handlers.ResponseStruct{Data=map[string]prometheus.RealTimeAlertRule}  "alertrule"
// @Router       /v1/proxy/cluster/{cluster}/custom/prometheus/v1/alertrule [get]
// @Security     JWT
func (p *prometheusHandler) AlertRule(c *gin.Context) {
	groupOrAlertName := c.Query("name")
	api := v1.NewAPI(p.client)
	rules, err := api.Rules(context.TODO())
	if err != nil {
		NotOK(c, err)
		return
	}

	// gems-namespace-name 作为key
	ret := make(map[string]prometheus.RealTimeAlertRule)
	// 按group聚合
	for _, g := range rules.Groups {
		if groupOrAlertName == "" || strings.Contains(g.Name, groupOrAlertName) {
			for _, r := range g.Rules {
				switch v := r.(type) {
				case v1.AlertingRule:
					namespace := v.Labels[prometheus.AlertNamespaceLabel]
					name := v.Labels[prometheus.AlertNameLabel]
					if namespace != "" && name != "" {
						key := prometheus.RealTimeAlertKey(string(namespace), string(name))
						if v.Name == g.Name {
							alert, ok := ret[key]
							if ok {
								alert.Alerts = append(alert.Alerts, v.Alerts...)
								alert.State = getState(alert.State, v.State)
							} else {
								alert = prometheus.RealTimeAlertRule{
									Alerts: v.Alerts,
									Name:   v.Name,
									State:  getState("", v.State),
								}
							}
							ret[key] = alert
						}
					}
				}
			}
		}
	}
	OK(c, ret)
}

const (
	// kube-proxy:
	// labels:
	// 		namespace="kube-system"
	// discoveredLabels:
	// 		__meta_kubernetes_namespace: "kube-system"
	// 		__meta_kubernetes_pod_controller_kind: "DaemonSet"
	// 		__meta_kubernetes_pod_controller_name: "kube-proxy"

	// kube-scheduler,kube-controller-manager:
	// labels:
	// 		namespace="kube-system"
	// discoveredLabels:
	// 		__meta_kubernetes_namespace: kube-system
	// 		__meta_kubernetes_pod_controller_kind: Node
	// 		__meta_kubernetes_pod_label_component: kube-controller-manager

	// apiserver:
	// labels:
	// 		job="kubernetes" || service="kubernetes"
	// discoveredLabels:
	//  	__meta_kubernetes_endpoints_name: kubernetes
	//  	__meta_kubernetes_service_name: kubernetes
	discoveredLabelsPodControllerKind = "__meta_kubernetes_pod_controller_kind"
	discoveredLabelsPodControllerName = "__meta_kubernetes_pod_controller_name"
	discoveredLabelsEndpointName      = "__meta_kubernetes_endpoints_name"
	discoveredLabelsServiceName       = "__meta_kubernetes_service_name"
	discoveredLabelsPodLabelComponent = "__meta_kubernetes_pod_label_component"
	labelNamespace                    = "namespace"
	labelsValueKubesystem             = "kube-system"
	labelValueKubernetes              = "kubernetes"
)

func isKubeScheduler(target v1.ActiveTarget) bool {
	return target.Labels[labelNamespace] == labelsValueKubesystem &&
		target.DiscoveredLabels[discoveredLabelsPodControllerKind] != "Node" &&
		target.DiscoveredLabels[discoveredLabelsPodLabelComponent] != "kube-scheduler"
}

func isKubeControllerManagerTarget(target v1.ActiveTarget) bool {
	return target.Labels[labelNamespace] == labelsValueKubesystem &&
		target.DiscoveredLabels[discoveredLabelsPodControllerKind] != "Node" &&
		target.DiscoveredLabels[discoveredLabelsPodLabelComponent] != "kube-controller-manager"
}

func isKubeProxyTarget(target v1.ActiveTarget) bool {
	return target.Labels[labelNamespace] == labelsValueKubesystem &&
		target.DiscoveredLabels[discoveredLabelsPodControllerKind] == "DaemonSet" &&
		target.DiscoveredLabels[discoveredLabelsPodControllerName] == "kube-proxy"
}

func isApiserverTarget(target v1.ActiveTarget) bool {
	if target.Labels["job"] == labelValueKubernetes || target.Labels["service"] == labelValueKubernetes {
		return true
	}
	if target.DiscoveredLabels[discoveredLabelsServiceName] == labelValueKubernetes ||
		target.DiscoveredLabels[discoveredLabelsEndpointName] == labelValueKubernetes {
		return true
	}
	return false
}

type ComponentStatus struct {
	IsHealthy bool
	Reasons   []string `json:",omitempty"`
	Count     int
}

// @Tags         Agent.V1
// @Summary      ComponentStatus
// @Description  ComponentStatus 获取集群组件状态
// @Accept       json
// @Produce      json
// @Success      200  {object}  handlers.ResponseStruct{Data=map[string]ComponentStatus}  "ComponentStatus"
// @Router       /v1/proxy/cluster/{cluster}/custom/prometheus/v1/componentstatus [get]
// @Security     JWT
func (p *prometheusHandler) ComponentStatus(c *gin.Context) {
	api := v1.NewAPI(p.client)
	targets, err := api.Targets(context.TODO())
	if err != nil {
		NotOK(c, err)
		return
	}

	ret := map[string]ComponentStatus{}

	// TODO etcd状态考虑用相同的方式获取，不过现在prometheus无法直接从etcd拿数据
	// 因为没有创建etcd证书的secret，只有从apiserver拿一小部分
	obj, _, err := api.Query(c.Request.Context(), "etcd_object_counts", time.Now())
	etcdstatus := ComponentStatus{
		IsHealthy: obj.String() != "",
	}
	if err != nil {
		etcdstatus.Reasons = []string{err.Error()}
	}
	if err == nil && obj.String() == "" {
		etcdstatus.Reasons = []string{"Can't collect etcd metrics"}
	}
	ret["ETCD"] = etcdstatus

	var apiserver, controller, scheduler, proxy ComponentStatus
	for _, v := range targets.Active {
		if isApiserverTarget(v) {
			apiserver.Count++
			if v.Health != v1.HealthGood {
				apiserver.Reasons = append(apiserver.Reasons, v.LastError)
				continue
			}
		}
		if isKubeControllerManagerTarget(v) {
			controller.Count++
			if v.Health != v1.HealthGood {
				controller.Reasons = append(controller.Reasons, v.LastError)
				continue
			}
		}
		if isKubeScheduler(v) {
			scheduler.Count++
			if v.Health != v1.HealthGood {
				scheduler.Reasons = append(scheduler.Reasons, v.LastError)
				continue
			}
		}
		if isKubeProxyTarget(v) {
			proxy.Count++
			if v.Health != v1.HealthGood {
				proxy.Reasons = append(proxy.Reasons, v.LastError)
				continue
			}
		}
	}

	if apiserver.Count > 0 {
		apiserver.IsHealthy = len(apiserver.Reasons) == 0
	} else {
		apiserver.IsHealthy = false
		apiserver.Reasons = []string{"APIServer not found!"}
	}
	if controller.Count > 0 {
		controller.IsHealthy = len(controller.Reasons) == 0
	} else {
		controller.IsHealthy = false
		controller.Reasons = []string{"ControllerManager not found!"}
	}
	if scheduler.Count > 0 {
		scheduler.IsHealthy = len(scheduler.Reasons) == 0
	} else {
		scheduler.IsHealthy = false
		scheduler.Reasons = []string{"Scheduler not found!"}
	}
	if proxy.Count > 0 {
		proxy.IsHealthy = len(proxy.Reasons) == 0
	} else {
		proxy.IsHealthy = false
		proxy.Reasons = []string{"Proxy not found!"}
	}
	ret["APIServer"] = apiserver
	ret["ControllerManager"] = controller
	ret["Scheduler"] = scheduler
	ret["Proxy"] = proxy
	OK(c, ret)
}

// @Tags         Agent.V1
// @Summary      CertInfo
// @Description  CertInfo 获取证书信息
// @Accept       json
// @Produce      json
// @Param        name  path      string                                           false  "name"
// @Success      200   {object}  handlers.ResponseStruct{Data=map[string]string}  "CertInfo"
// @Router       /v1/proxy/cluster/{cluster}/custom/prometheus/v1/certinfos/{name} [get]
// @Security     JWT
func (p *prometheusHandler) CertInfo(c *gin.Context) {
	if c.Param("name") == "apiserver" {
		expiredAt, err := clusterinfo.GetServerCertExpiredTime(clusterinfo.APIServerURL, clusterinfo.APIServerCertCN)
		if err != nil {
			NotOK(c, err)
			return
		}
		OK(c, gin.H{
			"ExpiredAt": expiredAt,
		})
	} else {
		NotOK(c, fmt.Errorf("unsupport cert name"))
		return
	}
}

func getState(old, new string) string {
	if stateMap[new] > stateMap[old] {
		return new
	}
	return old
}
