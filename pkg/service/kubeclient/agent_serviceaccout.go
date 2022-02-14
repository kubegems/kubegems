package kubeclient

import (
	v1 "k8s.io/api/core/v1"
)

var (
	serviceAccountGVKi = v1.SchemeGroupVersion.WithKind("ServiceAccount")
	serviceAccountGVK  = &serviceAccountGVKi
)

func (k KubeClient) GetServiceAccount(cluster, namespace, name string, labels map[string]string) (*v1.ServiceAccount, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1.ServiceAccount{}
	err = agentClient.GetObject(serviceAccountGVK, obj, &namespace, &name)
	return obj, err
}

func (k KubeClient) GetServiceAccountList(cluster, namespace string, labelSet map[string]string) (*[]v1.ServiceAccount, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1.ServiceAccount{}
	err = agentClient.GetObjectList(serviceAccountGVK, &list, &namespace, labelSet)
	return &list, err
}

func (k KubeClient) DeleteServiceAccount(cluster, namespace, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(serviceAccountGVK, &namespace, &name)
}

func (k KubeClient) CreateServiceAccount(cluster, namespace, name string, data *v1.ServiceAccount) (*v1.ServiceAccount, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.CreateObject(serviceAccountGVK, data, &namespace, &name)
	return data, err
}

func (k KubeClient) PatchServiceAccount(cluster, namespace, name string, data *v1.ServiceAccount) (*v1.ServiceAccount, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.PatchObject(serviceAccountGVK, data, &namespace, &name)
	return data, err
}
