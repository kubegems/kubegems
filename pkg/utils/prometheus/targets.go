package prometheus

import (
	"fmt"
	"strings"

	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/apis/gems"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MetricTargetService     = "service"
	MetricTargetDeployment  = "deployment"
	MetricTargetStatefulset = "statefulset"
	MetricTargetDaemonset   = "daemonset"
	MetricTargetWorkload    = "workload" // 临时解决不知道哪种workload的情况

	allNamespace = "_all"
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

func (t *MetricTarget) GetMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: t.Namespace,
		Name:      t.Name,
		Annotations: map[string]string{
			gems.LabelMonitorCollector: t.TargetType + "-" + t.TargetName,
		},
	}
}

func (t *MetricTarget) toServiceMonitor() *v1.ServiceMonitor {
	ret := &v1.ServiceMonitor{
		ObjectMeta: t.GetMeta(),
	}
	f := MutateServiceMonitorFunc(t, ret)
	f()
	return ret
}

func (t *MetricTarget) toPodMonitor() *v1.PodMonitor {
	ret := &v1.PodMonitor{
		ObjectMeta: t.GetMeta(),
	}
	f := MutatePodMonitorFunc(t, ret)
	f()
	return ret
}

func ConvertToServiceOrPodMonitor(req *MetricTarget) (client.Object, error) {
	switch req.TargetType {
	case MetricTargetService:
		return req.toServiceMonitor(), nil
	case MetricTargetDeployment, MetricTargetStatefulset, MetricTargetDaemonset, MetricTargetWorkload:
		return req.toPodMonitor(), nil
	}
	return nil, fmt.Errorf("not valid metric target type, must be one of service/deployment/statefulset/daemonset/workload")
}

func ConvertToMetricTarget(obj client.Object) *MetricTarget {
	switch v := obj.(type) {
	case *v1.ServiceMonitor:
		ret := &MetricTarget{
			Namespace:       v.Namespace,
			Name:            v.Name,
			TargetType:      MetricTargetService,
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
		if ret.TargetType == "" {
			ret.TargetType = MetricTargetWorkload
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

func MutateServiceMonitorFunc(t *MetricTarget, sm *v1.ServiceMonitor) func() error {
	return func() error {
		sm.Spec = v1.ServiceMonitorSpec{
			Selector:          *metav1.SetAsLabelSelector(t.TargetLabels),
			NamespaceSelector: NamespaceSelector(t.TargetNamespace, t.Namespace),
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

func MutatePodMonitorFunc(t *MetricTarget, pm *v1.PodMonitor) func() error {
	return func() error {
		pm.Spec = v1.PodMonitorSpec{
			Selector:            *metav1.SetAsLabelSelector(t.TargetLabels),
			NamespaceSelector:   NamespaceSelector(t.TargetNamespace, t.Namespace),
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
		tmp := strings.SplitN(anno[gems.LabelMonitorCollector], "-", 2)
		if len(tmp) > 1 {
			return tmp[0]
		}
	}
	return ""
}

func getTargetNameFromAnno(anno map[string]string) string {
	if anno != nil {
		tmp := strings.SplitN(anno[gems.LabelMonitorCollector], "-", 2)
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
func NamespaceSelector(ns string, def string) v1.NamespaceSelector {
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
