package kubeclient

import (
	"net/http"

	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type NodeMetricsResponse struct {
	Message   string
	Data      *[]v1beta1.NodeMetrics
	ErrorData string
}

type PodsMetricsResponse struct {
	Message   string
	Data      *[]v1beta1.PodMetrics
	ErrorData string
}

func (k *KubeClient) GetNodeMetrics(cluster string) (*[]v1beta1.NodeMetrics, error) {
	url := formatURL(nil, nil, nil, "Metric/Node")
	ret := &[]v1beta1.NodeMetrics{}
	if err := k.request(http.MethodGet, cluster, url, nil, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (k *KubeClient) GetPodsMetrics(cluster, namespace string) (*[]v1beta1.PodMetrics, error) {
	url := formatURL(map[string]string{"namespace": namespace}, nil, nil, "Metric/Pod/{namespace}")
	ret := &[]v1beta1.PodMetrics{}
	if err := k.request(http.MethodGet, cluster, url, nil, ret); err != nil {
		return nil, err
	}
	return ret, nil
}
