package kubeclient

import (
	v1 "k8s.io/api/core/v1"
)

var (
	serviceGVKi = v1.SchemeGroupVersion.WithKind("Service")
	serviceGVK  = &serviceGVKi
)

func (k KubeClient) GetService(cluster, namespace, name string, labels map[string]string) (*v1.Service, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1.Service{}
	err = agentClient.GetObject(serviceGVK, obj, &namespace, &name)
	return obj, err
}

func (k KubeClient) GetServiceList(cluster, namespace string, labelSet map[string]string) (*[]v1.Service, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1.Service{}
	err = agentClient.GetObjectList(serviceGVK, &list, &namespace, labelSet)
	return &list, err
}

func (k KubeClient) DeleteService(cluster, namespace, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(serviceGVK, &namespace, &name)
}

func (k KubeClient) CreateService(cluster, namespace, name string, data *v1.Service) (*v1.Service, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.CreateObject(serviceGVK, data, &namespace, &name)
	return data, err
}

func (k KubeClient) PatchService(cluster, namespace, name string, data *v1.Service) (*v1.Service, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.PatchObject(serviceGVK, data, &namespace, &name)
	return data, err
}
