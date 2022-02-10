package kubeclient

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/kubegems/gems/pkg/log"
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/common/model"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var prometheusRuleGVK = &schema.GroupVersionKind{Group: "monitoring.coreos.com", Kind: "prometheusrules", Version: "v1"}

func (k KubeClient) GetPrometheusRule(cluster, namespace, name string, labels map[string]string) (*v1.PrometheusRule, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1.PrometheusRule{}
	err = agentClient.GetObject(prometheusRuleGVK, obj, &namespace, &name)
	return obj, err
}

func (k KubeClient) GetPrometheusRuleList(cluster, namespace string, labelSet map[string]string) (*[]*v1.PrometheusRule, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []*v1.PrometheusRule{}
	err = agentClient.GetObjectList(prometheusRuleGVK, &list, &namespace, labelSet)
	return &list, err
}

func (k KubeClient) DeletePrometheusRule(cluster, namespace, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(prometheusRuleGVK, &namespace, &name)
}

func (k KubeClient) CreatePrometheusRule(cluster, namespace, name string, data *v1.PrometheusRule) (*v1.PrometheusRule, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	if err := checkPrometheusRule(data); err != nil {
		return nil, err
	}
	err = agentClient.CreateObject(prometheusRuleGVK, data, &namespace, &name)
	return data, err
}

func (k KubeClient) UpdatePrometheusRule(cluster string, data *v1.PrometheusRule) (*v1.PrometheusRule, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	if err := checkPrometheusRule(data); err != nil {
		return nil, err
	}
	name := data.GetName()
	ns := data.GetNamespace()
	err = agentClient.UpdateObject(prometheusRuleGVK, data, &ns, &name)
	return data, err
}

func checkPrometheusRule(rule *v1.PrometheusRule) error {
	for _, g := range rule.Spec.Groups {
		for _, r := range g.Rules {
			if r.Alert == "" {
				return fmt.Errorf("rule must be alert")
			}
			if r.Record != "" {
				return fmt.Errorf("rule can't be record")
			}
		}
	}
	return nil
}

func (k KubeClient) PrometheusQueryRange(cluster, promql string, start, end string, step string) (model.Matrix, error) {
	ret := model.Matrix{}

	values := url.Values{}
	values.Add("query", promql)
	values.Add("start", start)
	values.Add("end", end)
	values.Add("step", step)
	err := k.DoRequest(http.MethodGet, cluster, fmt.Sprintf("/custom/prometheus/v1/matrix?%s", values.Encode()), nil, &ret)
	if err != nil {
		log.Error(err, "prometheus query range", "cluster", cluster, "promql", promql)
	}
	return ret, err
}
