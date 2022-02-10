package kubeclient

import (
	"github.com/kubegems/gems/pkg/apis/gems/v1beta1"
)

func (k KubeClient) GetEnvironment(cluster, name string, _ map[string]string) (*v1beta1.Environment, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1beta1.Environment{}
	err = agentClient.GetObject(&v1beta1.SchemeEnvironment, obj, nil, &name)
	return obj, err
}

func (k KubeClient) GetEnvironmentList(cluster string, labelSet map[string]string) (*[]v1beta1.Environment, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1beta1.Environment{}
	err = agentClient.GetObjectList(&v1beta1.SchemeEnvironment, &list, nil, labelSet)
	return &list, err
}

func (k KubeClient) DeleteEnvironment(cluster, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(&v1beta1.SchemeEnvironment, nil, &name)
}

func (k KubeClient) CreateEnvironment(cluster, name string, data *v1beta1.Environment) (*v1beta1.Environment, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.CreateObject(&v1beta1.SchemeEnvironment, data, nil, &name)
	return data, err
}

func (k KubeClient) PatchEnvironment(cluster, name string, data *v1beta1.Environment) (*v1beta1.Environment, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.PatchObject(&v1beta1.SchemeEnvironment, data, nil, &name)
	return data, err
}

func (k KubeClient) CreateOrUpdateEnvironment(clustername, environment string, spec v1beta1.EnvironmentSpec) error {
	env := &v1beta1.Environment{}
	exist, err := GetClient().GetEnvironment(clustername, environment, nil)
	if err != nil {
		env.Name = environment
		env.Spec = spec
		_, e := GetClient().CreateEnvironment(clustername, environment, env)
		return e
	}
	exist.Spec = spec
	exist.ObjectMeta.ResourceVersion = ""
	_, e := GetClient().PatchEnvironment(clustername, environment, exist)
	return e
}
