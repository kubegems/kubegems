package kubeclient

import (
	"k8s.io/api/storage/v1beta1"
)

var (
	scGVKi = v1beta1.SchemeGroupVersion.WithKind("StorageClass")
	scGVK  = &scGVKi
)

func (k KubeClient) GetStorageClass(cluster, name string, _ map[string]string) (*v1beta1.StorageClass, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1beta1.StorageClass{}
	err = agentClient.GetObject(scGVK, obj, nil, &name)
	return obj, err
}

func (k KubeClient) GetStorageClassList(cluster string, labelSet map[string]string) (*[]v1beta1.StorageClass, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1beta1.StorageClass{}
	err = agentClient.GetObjectList(scGVK, &list, nil, labelSet)
	return &list, err
}

func (k KubeClient) DeleteStorageClass(cluster, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(scGVK, nil, &name)
}

func (k KubeClient) CreateStorageClass(cluster, name string, data *v1beta1.StorageClass) (*v1beta1.StorageClass, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.CreateObject(scGVK, data, nil, &name)
	return data, err
}

func (k KubeClient) PatchStorageClass(cluster, name string, data *v1beta1.StorageClass) (*v1beta1.StorageClass, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.PatchObject(scGVK, data, nil, &name)
	return data, err
}
