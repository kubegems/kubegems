package kubeclient

import (
	"istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

func (k KubeClient) GetIstioOperator(cluster, namespace, name string, labels map[string]string) (*v1alpha1.IstioOperator, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1alpha1.IstioOperator{}
	err = agentClient.GetObject(&v1alpha1.IstioOperatorGVK, obj, &namespace, &name)
	return obj, err
}

func (k KubeClient) GetIstioOperatorList(cluster, namespace string, labelSet map[string]string) (*[]v1alpha1.IstioOperator, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1alpha1.IstioOperator{}
	err = agentClient.GetObjectList(&v1alpha1.IstioOperatorGVK, &list, &namespace, labelSet)
	return &list, err
}

func (k KubeClient) DeleteIstioOperator(cluster, namespace, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(&v1alpha1.IstioOperatorGVK, &namespace, &name)
}

func (k KubeClient) CreateIstioOperator(cluster, namespace, name string, data *v1alpha1.IstioOperator) (*v1alpha1.IstioOperator, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.CreateObject(&v1alpha1.IstioOperatorGVK, data, &namespace, &name)
	return data, err
}

func (k KubeClient) UpdateIstioOperator(cluster, namespace, name string, data *v1alpha1.IstioOperator) (*v1alpha1.IstioOperator, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.UpdateObject(&v1alpha1.IstioOperatorGVK, data, &namespace, &name)
	return data, err
}
