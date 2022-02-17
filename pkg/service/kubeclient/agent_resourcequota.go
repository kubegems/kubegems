package kubeclient

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var rqGVK = &schema.GroupVersionKind{Group: "core", Kind: "resourcequotas", Version: "v1"}

func (k KubeClient) GetResourceQuota(cluster, namespace, name string, labels map[string]string) (*v1.ResourceQuota, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1.ResourceQuota{}
	err = agentClient.GetObject(rqGVK, obj, &namespace, &name)
	return obj, err
}

func (k KubeClient) GetResourceQuotaList(cluster, namespace string, labelSet map[string]string) (*[]v1.ResourceQuota, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1.ResourceQuota{}
	err = agentClient.GetObjectList(rqGVK, &list, &namespace, labelSet)
	return &list, err
}

func (k KubeClient) DeleteResourceQuota(cluster, namespace, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(rqGVK, &namespace, &name)
}
