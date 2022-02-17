package kubeclient

import (
	v1 "k8s.io/api/core/v1"
)

var (
	secretGVKi = v1.SchemeGroupVersion.WithKind("Secret")
	secretGVK  = &secretGVKi
)

func (k KubeClient) GetSecret(cluster, namespace, name string, labels map[string]string) (*v1.Secret, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1.Secret{}
	err = agentClient.GetObject(secretGVK, obj, &namespace, &name)
	return obj, err
}

func (k KubeClient) DeleteSecret(cluster, namespace, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(secretGVK, &namespace, &name)
}

func (k KubeClient) CreateSecret(cluster, namespace, name string, data *v1.Secret) (*v1.Secret, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.CreateObject(secretGVK, data, &namespace, &name)
	return data, err
}

func (k KubeClient) PatchSecret(cluster, namespace, name string, data *v1.Secret) (*v1.Secret, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.PatchObject(secretGVK, data, &namespace, &name)
	return data, err
}

func (k KubeClient) CreateOrUpdateSecret(clustername, namespace, name string, data map[string][]byte) error {
	exist, err := GetClient().GetSecret(clustername, namespace, name, nil)
	if err != nil {
		sec := &v1.Secret{}
		sec.Name = name
		sec.Namespace = namespace
		sec.Type = v1.SecretTypeDockerConfigJson
		sec.Data = data
		_, e := GetClient().CreateSecret(clustername, namespace, name, sec)
		return e
	}
	exist.Data = data
	exist.ObjectMeta.ResourceVersion = ""
	_, e := GetClient().PatchSecret(clustername, namespace, name, exist)
	return e
}

func (k KubeClient) DeleteSecretIfExist(clustername, namespace, name string) error {
	_, err := GetClient().GetSecret(clustername, namespace, name, nil)
	if err != nil {
		return nil
	}
	return GetClient().DeleteSecret(clustername, namespace, name)
}
