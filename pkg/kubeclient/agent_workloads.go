package kubeclient

import (
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	deployGVK = &schema.GroupVersionKind{Group: "core", Kind: "deployments", Version: "v1"}
	stsGVK    = &schema.GroupVersionKind{Group: "core", Kind: "statefulsets", Version: "v1"}
	dsGVK     = &schema.GroupVersionKind{Group: "core", Kind: "daemonsets", Version: "v1"}
)

func (k KubeClient) ListDeploymentByExistLabels(cluster, namespace string, label []string) ([]v1.Deployment, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1.Deployment{}
	labels := map[string]string{}
	for _, k := range label {
		labels[k+"__exist"] = "_"
	}
	err = agentClient.GetObjectList(deployGVK, &list, &namespace, labels)
	return list, err
}

func (k KubeClient) ListStatefulsetByExistLabels(cluster, namespace string, label []string) ([]v1.StatefulSet, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1.StatefulSet{}
	labels := map[string]string{}
	for _, k := range label {
		labels[k+"__exist"] = "_"
	}
	err = agentClient.GetObjectList(stsGVK, &list, &namespace, labels)
	return list, err
}

func (k KubeClient) ListDaemonsetByExistLabels(cluster, namespace string, label []string) ([]v1.DaemonSet, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1.DaemonSet{}
	labels := map[string]string{}
	for _, k := range label {
		labels[k+"__exist"] = "_"
	}
	err = agentClient.GetObjectList(dsGVK, &list, &namespace, labels)
	return list, err
}
