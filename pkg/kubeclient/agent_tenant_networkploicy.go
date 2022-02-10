package kubeclient

import (
	"github.com/kubegems/gems/pkg/apis/gems/v1beta1"
)

func (k KubeClient) GetTenantNetworkPolicy(cluster, name string, _ map[string]string) (*v1beta1.TenantNetworkPolicy, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1beta1.TenantNetworkPolicy{}
	err = agentClient.GetObject(&v1beta1.SchemeTenantNetworkPolicy, obj, nil, &name)
	return obj, err
}

func (k KubeClient) GetTenantNetworkPolicyList(cluster string, labelSet map[string]string) (*[]v1beta1.TenantNetworkPolicy, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1beta1.TenantNetworkPolicy{}
	err = agentClient.GetObjectList(&v1beta1.SchemeTenantNetworkPolicy, &list, nil, labelSet)
	return &list, err
}

func (k KubeClient) DeleteTenantNetworkPolicy(cluster, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(&v1beta1.SchemeTenantNetworkPolicy, nil, &name)
}

func (k KubeClient) CreateTenantNetworkPolicy(cluster, name string, data *v1beta1.TenantNetworkPolicy) (*v1beta1.TenantNetworkPolicy, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.CreateObject(&v1beta1.SchemeTenantNetworkPolicy, data, nil, &name)
	return data, err
}

func (k KubeClient) PatchTenantNetworkPolicy(cluster, name string, data *v1beta1.TenantNetworkPolicy) (*v1beta1.TenantNetworkPolicy, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.PatchObject(&v1beta1.SchemeTenantNetworkPolicy, data, nil, &name)
	return data, err
}
