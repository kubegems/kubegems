package kubeclient

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var nodeGVK = &schema.GroupVersionKind{Group: "core", Kind: "nodes", Version: "v1"}

func (k KubeClient) GetNode(cluster, name string, _ map[string]string) (*v1.Node, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1.Node{}
	err = agentClient.GetObject(nodeGVK, obj, nil, &name)
	return obj, err
}

func (k KubeClient) GetNodeList(cluster string, labelSet map[string]string) (*[]v1.Node, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1.Node{}
	err = agentClient.GetObjectList(nodeGVK, &list, nil, labelSet)
	return &list, err
}

func (k KubeClient) DeleteNode(cluster, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(nodeGVK, nil, &name)
}

func (k KubeClient) CreateNode(cluster, name string, data *v1.Node) (*v1.Node, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.CreateObject(nodeGVK, data, nil, &name)
	return data, err
}

func (k KubeClient) UpdateNode(cluster, name string, data *v1.Node) (*v1.Node, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.UpdateObject(nodeGVK, data, nil, &name)
	return data, err
}

func (k KubeClient) PatchNode(cluster, name string, data *v1.Node) (*v1.Node, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.PatchObject(nodeGVK, data, nil, &name)
	return data, err
}
