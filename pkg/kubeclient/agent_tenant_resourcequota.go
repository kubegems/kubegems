package kubeclient

import (
	"kubegems.io/pkg/apis/gems/v1beta1"
)

func (k KubeClient) GetTenantResourceQuota(cluster, name string, _ map[string]string) (*v1beta1.TenantResourceQuota, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1beta1.TenantResourceQuota{}
	err = agentClient.GetObject(&v1beta1.SchemeTenantResourceQuota, obj, nil, &name)
	return obj, err
}

func (k KubeClient) GetTenantResourceQuotaList(cluster string, labelSet map[string]string) (*[]v1beta1.TenantResourceQuota, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1beta1.TenantResourceQuota{}
	err = agentClient.GetObjectList(&v1beta1.SchemeTenantResourceQuota, &list, nil, labelSet)
	return &list, err
}

func (k KubeClient) DeleteTenantResourceQuota(cluster, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(&v1beta1.SchemeTenantResourceQuota, nil, &name)
}

func (k KubeClient) CreateTenantResourceQuota(cluster, name string, data *v1beta1.TenantResourceQuota) (*v1beta1.TenantResourceQuota, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.CreateObject(&v1beta1.SchemeTenantResourceQuota, data, nil, &name)
	return data, err
}

func (k KubeClient) PatchTenantResourceQuota(cluster, name string, data *v1beta1.TenantResourceQuota) (*v1beta1.TenantResourceQuota, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.PatchObject(&v1beta1.SchemeTenantResourceQuota, data, nil, &name)
	return data, err
}
