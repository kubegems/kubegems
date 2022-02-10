package kubeclient

import "net/http"

func (k KubeClient) ClusterWorkloadStatistics(cluster string, ret interface{}) error {
	url := "/custom/statistics.system/v1/workloads"
	if err := k.request(http.MethodGet, cluster, url, nil, ret); err != nil {
		return err
	}
	return nil
}

func (k KubeClient) ClusterResourceStatistics(cluster string, ret interface{}) error {
	url := "/custom/statistics.system/v1/resources"
	if err := k.request(http.MethodGet, cluster, url, nil, ret); err != nil {
		return err
	}
	return nil
}
