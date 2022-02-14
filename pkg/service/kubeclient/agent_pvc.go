package kubeclient

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var pvcGVK = &schema.GroupVersionKind{Group: "core", Kind: "persistentvolumeclaims", Version: "v1"}

func (k KubeClient) GetPersistentVolumeClaim(cluster, namespace, name string, labels map[string]string) (*v1.PersistentVolumeClaim, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1.PersistentVolumeClaim{}
	err = agentClient.GetObject(pvcGVK, obj, &namespace, &name)
	return obj, err
}

func (k KubeClient) GetPersistentVolumeClaimList(cluster, namespace string, labelSet map[string]string) (*[]v1.PersistentVolumeClaim, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1.PersistentVolumeClaim{}
	err = agentClient.GetObjectList(pvcGVK, &list, &namespace, labelSet)
	return &list, err
}

func (k KubeClient) DeletePersistentVolumeClaim(cluster, namespace, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(pvcGVK, &namespace, &name)
}

func (k KubeClient) CreatePersistentVolumeClaim(cluster, namespace, name string, data *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.CreateObject(pvcGVK, data, &namespace, &name)
	return data, err
}

func (k KubeClient) PatchPersistentVolumeClaim(cluster, namespace, name string, data *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.PatchObject(pvcGVK, data, &namespace, &name)
	return data, err
}
