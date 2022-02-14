package kubeclient

import (
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
)

var (
	hpaGVKi = v2beta1.SchemeGroupVersion.WithKind("HorizontalPodAutoscaler")
	hpaGVK  = &hpaGVKi
)

func (k KubeClient) GetHPA(cluster, namespace, name string) (*v2beta1.HorizontalPodAutoscaler, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v2beta1.HorizontalPodAutoscaler{}
	err = agentClient.GetObject(hpaGVK, obj, &namespace, &name)
	return obj, err
}

func (k KubeClient) CreateHPA(cluster string, data *v2beta1.HorizontalPodAutoscaler, namespace, name string) (*v2beta1.HorizontalPodAutoscaler, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v2beta1.HorizontalPodAutoscaler{}
	err = agentClient.CreateObject(hpaGVK, obj, &namespace, &name)
	return obj, err
}

func (k KubeClient) UpdateHPA(cluster string, data *v2beta1.HorizontalPodAutoscaler, namespace, name string) (*v2beta1.HorizontalPodAutoscaler, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	err = agentClient.UpdateObject(hpaGVK, data, &namespace, &name)
	return data, err
}

func (k KubeClient) GetOrCreateHPA(cluster string, data *v2beta1.HorizontalPodAutoscaler, namespace, name string) (*v2beta1.HorizontalPodAutoscaler, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	origin := &v2beta1.HorizontalPodAutoscaler{}
	err = agentClient.GetObject(hpaGVK, origin, &namespace, &name)
	if err == nil {
		return origin, err
	}
	err = agentClient.CreateObject(hpaGVK, data, &namespace, &name)
	return data, err
}

func (k KubeClient) CreateOrUpdateHPA(cluster string, data *v2beta1.HorizontalPodAutoscaler, namespace, name string) (*v2beta1.HorizontalPodAutoscaler, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	origin := &v2beta1.HorizontalPodAutoscaler{}
	err = agentClient.GetObject(hpaGVK, origin, &namespace, &name)
	if err == nil {
		err = agentClient.UpdateObject(hpaGVK, data, &namespace, &name)
		return data, err
	} else {
		err = agentClient.CreateObject(hpaGVK, data, &namespace, &name)
		return data, err
	}
}
