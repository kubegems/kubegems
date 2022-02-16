package metrics

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/kubeclient"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/pagination"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type MetricTarget struct {
	Cluster   string `json:"-"`
	Namespace string `json:"namespace"` // 采集器所在namespace
	Name      string `json:"name"`      // 采集器名，前端默认构造为[{name}-{service/deployment/...}-metrics]

	TargetType      string            `json:"target_type"`      // 采集器类型, service/deployment/statefulset/daemonset
	TargetNamespace string            `json:"target_namespace"` // 要采集的目标namespace, 支持所有namespace: _all
	TargetName      string            `json:"target_name"`      // 要采集的service/deployment/statefulset/daemonset名
	TargetLabels    map[string]string `json:"target_labels"`    // 标签筛选

	TargetEndpoints []TargetEndpoint `json:"target_endpoints"`
}

type TargetEndpoint struct {
	Port        string `json:"port"`         // 端口名
	HonorLabels bool   `json:"honor_labels"` // 是否优先选用原生标签
	Interval    string `json:"interval"`     // 多久采集一次, default 30s
	Path        string `json:"path"`         // 采集路径, default: /metrics
}

func (t MetricTarget) GetName() string {
	return t.Name
}

func (t MetricTarget) GetCreationTimestamp() metav1.Time {
	return metav1.Time{}
}

func (t *MetricTarget) getMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: t.Namespace,
		Name:      t.Name,
		Annotations: map[string]string{
			annotationsTargetNameKey: t.TargetType + "-" + t.TargetName,
		},
	}
}

func (t *MetricTarget) toServiceMonitor() *v1.ServiceMonitor {
	ret := &v1.ServiceMonitor{
		ObjectMeta: t.getMeta(),
	}
	f := mutateServiceMonitorFunc(t, ret)
	f()
	return ret
}

func (t *MetricTarget) toPodMonitor() *v1.PodMonitor {
	ret := &v1.PodMonitor{
		ObjectMeta: t.getMeta(),
	}
	f := mutatePodMonitorFunc(t, ret)
	f()
	return ret
}

const (
	metricTargetService     = "service"
	metricTargetDeployment  = "deployment"
	metricTargetStatefulset = "statefulset"
	metricTargetDaemonset   = "daemonset"

	annotationsTargetNameKey = gems.AnnotationsMetricsTargetNameKey
	allNamespace             = "_all"
)

// ListMetricTarget 采集器列表
// @Tags Metrics
// @Summary  采集器列表
// @Description 采集器列表
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Success 200 {object} handlers.ResponseStruct{Data=[]MetricTarget} "resp"
// @Router /v1/metrics/cluster/{cluster}/namespaces/{namespace}/targets [get]
// @Security JWT
func (h *MonitorHandler) ListMetricTarget(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	if namespace == allNamespace {
		namespace = corev1.NamespaceAll
	}
	ctx := c.Request.Context()
	pms := v1.PodMonitorList{}
	sms := v1.ServiceMonitorList{}
	if err := kubeclient.Execute(ctx, cluster, func(tc agents.Client) error {
		g := errgroup.Group{}
		g.Go(func() error {
			return tc.List(ctx, &pms, client.InNamespace(namespace))
		})
		g.Go(func() error {
			return tc.List(ctx, &sms, client.InNamespace(namespace))
		})
		return g.Wait()
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}

	ret := []*MetricTarget{}
	for _, v := range pms.Items {
		ret = append(ret, convertToMetricTarget(v))
	}
	for _, v := range sms.Items {
		ret = append(ret, convertToMetricTarget(v))
	}

	sort.Slice(ret, func(i, j int) bool {
		return strings.ToLower(ret[i].Name) < strings.ToLower(ret[j].Name)
	})
	handlers.OK(c, pagination.NewPageDataFromContextReflect(c, ret))
}

// AddOrUpdateMetricTarget 添加/更新采集器
// @Tags Metrics
// @Summary  添加/更新采集器
// @Description 添加/更新采集器
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param form body MetricTarget true "采集器内容"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "resp"
// @Router /v1/metrics/cluster/{cluster}/namespaces/{namespace}/targets [post]
// @Security JWT
func (h *MonitorHandler) AddOrUpdateMetricTarget(c *gin.Context) {
	if err := withMetricTargetReq(c, func(req *MetricTarget) error {
		ctx := c.Request.Context()
		tc, err := h.GetAgentsClientSet().ClientOf(ctx, req.Cluster)
		if err != nil {
			return err
		}

		switch req.TargetType {
		case metricTargetService:
			sm := &v1.ServiceMonitor{
				ObjectMeta: req.getMeta(),
			}
			_, err := controllerutil.CreateOrUpdate(ctx, tc, sm, mutateServiceMonitorFunc(req, sm))
			return err
		case metricTargetDeployment, metricTargetStatefulset, metricTargetDaemonset:
			pm := &v1.PodMonitor{
				ObjectMeta: req.getMeta(),
			}
			_, err := controllerutil.CreateOrUpdate(ctx, tc, pm, mutatePodMonitorFunc(req, pm))
			return err
		}
		return nil
	}); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

// DeleteMetricTarget 删除采集器
// @Tags Metrics
// @Summary  删除采集器
// @Description 删除采集器
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "采集器名"
// @Param type query string true "采集器类型, service/deployment/statefulset/daemonset"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "resp"
// @Router /v1/metrics/cluster/{cluster}/namespaces/{namespace}/targets/{name} [delete]
// @Security JWT
func (h *MonitorHandler) DeleteMetricTarget(c *gin.Context) {
	req := &MetricTarget{
		Cluster:      c.Param("cluster"),
		Namespace:    c.Param("namespace"),
		Name:         c.Param("name"),
		TargetType:   c.Query("type"),
		TargetLabels: make(map[string]string), // 避免空指针
	}
	obj, err := convertToServiceOrPodMonitor(req)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	ctx := c.Request.Context()
	tc, err := h.GetAgentsClientSet().ClientOf(ctx, req.Cluster)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := tc.Delete(ctx, obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

func withMetricTargetReq(c *gin.Context, f func(req *MetricTarget) error) error {
	req := MetricTarget{}
	if err := c.BindJSON(&req); err != nil {
		return err
	}
	// 以url上的c为准
	req.Cluster = c.Param("cluster")
	req.Namespace = c.Param("namespace")
	return f(&req)
}

func convertToServiceOrPodMonitor(req *MetricTarget) (client.Object, error) {
	switch req.TargetType {
	case metricTargetService:
		return req.toServiceMonitor(), nil
	case metricTargetDeployment, metricTargetStatefulset, metricTargetDaemonset:
		return req.toPodMonitor(), nil
	}
	return nil, fmt.Errorf("not valid metric target type, must be one of service/deployment/statefulset/daemonset")
}

func convertToMetricTarget(obj client.Object) *MetricTarget {
	switch v := obj.(type) {
	case *v1.ServiceMonitor:
		ret := &MetricTarget{
			Namespace:       v.Namespace,
			Name:            v.Name,
			TargetType:      getTargetTypeFromAnno(v.Annotations),
			TargetLabels:    v.Spec.Selector.MatchLabels,
			TargetNamespace: targetNamespace(v.Spec.NamespaceSelector),
			TargetName:      getTargetNameFromAnno(v.Annotations),
			TargetEndpoints: make([]TargetEndpoint, len(v.Spec.Endpoints)),
		}
		for i, ep := range v.Spec.Endpoints {
			ret.TargetEndpoints[i] = TargetEndpoint{
				Port:        ep.Port,
				HonorLabels: ep.HonorLabels,
				Interval:    ep.Interval,
				Path:        ep.Path,
			}
		}
		return ret
	case *v1.PodMonitor:
		ret := &MetricTarget{
			Namespace:       v.Namespace,
			Name:            v.Name,
			TargetType:      getTargetTypeFromAnno(v.Annotations),
			TargetLabels:    v.Spec.Selector.MatchLabels,
			TargetNamespace: targetNamespace(v.Spec.NamespaceSelector),
			TargetName:      getTargetNameFromAnno(v.Annotations),
			TargetEndpoints: make([]TargetEndpoint, len(v.Spec.PodMetricsEndpoints)),
		}
		for i, ep := range v.Spec.PodMetricsEndpoints {
			ret.TargetEndpoints[i] = TargetEndpoint{
				Port:        ep.Port,
				HonorLabels: ep.HonorLabels,
				Interval:    ep.Interval,
				Path:        ep.Path,
			}
		}
		return ret
	}
	return nil
}

func mutateServiceMonitorFunc(t *MetricTarget, sm *v1.ServiceMonitor) func() error {
	return func() error {
		sm.Spec = v1.ServiceMonitorSpec{
			Selector:          *metav1.SetAsLabelSelector(t.TargetLabels),
			NamespaceSelector: namespaceSelector(t.TargetNamespace, t.Namespace),
			Endpoints:         make([]v1.Endpoint, len(t.TargetEndpoints)),
		}
		for i, ep := range t.TargetEndpoints {
			sm.Spec.Endpoints[i] = v1.Endpoint{
				Port:        ep.Port,
				HonorLabels: ep.HonorLabels,
				Interval:    ep.Interval,
				Path:        metricPath(ep.Path),
			}
		}
		return nil
	}
}

func mutatePodMonitorFunc(t *MetricTarget, pm *v1.PodMonitor) func() error {
	return func() error {
		pm.Spec = v1.PodMonitorSpec{
			Selector:            *metav1.SetAsLabelSelector(t.TargetLabels),
			NamespaceSelector:   namespaceSelector(t.TargetNamespace, t.Namespace),
			PodMetricsEndpoints: make([]v1.PodMetricsEndpoint, len(t.TargetEndpoints)),
		}
		for i, ep := range t.TargetEndpoints {
			pm.Spec.PodMetricsEndpoints[i] = v1.PodMetricsEndpoint{
				Port:        ep.Port,
				HonorLabels: ep.HonorLabels,
				Interval:    ep.Interval,
				Path:        metricPath(ep.Path),
			}
		}
		return nil
	}
}

func getTargetTypeFromAnno(anno map[string]string) string {
	if anno != nil {
		tmp := strings.SplitN(anno[annotationsTargetNameKey], "-", 2)
		if len(tmp) > 1 {
			return tmp[0]
		}
	}
	return ""
}

func getTargetNameFromAnno(anno map[string]string) string {
	if anno != nil {
		tmp := strings.SplitN(anno[annotationsTargetNameKey], "-", 2)
		if len(tmp) > 1 {
			return tmp[1]
		}
	}
	return ""
}

func targetNamespace(s v1.NamespaceSelector) string {
	if s.Any {
		return allNamespace
	}
	if len(s.MatchNames) > 0 {
		return s.MatchNames[0]
	}
	return ""
}

// 默认当前ns
func namespaceSelector(ns string, def string) v1.NamespaceSelector {
	if ns == "" {
		ns = def
	}
	if ns == allNamespace {
		return v1.NamespaceSelector{
			Any: true,
		}
	} else {
		return v1.NamespaceSelector{
			Any:        false,
			MatchNames: []string{ns},
		}
	}
}

func metricPath(path string) string {
	if path == "" {
		return "/metrics"
	}
	return path
}
