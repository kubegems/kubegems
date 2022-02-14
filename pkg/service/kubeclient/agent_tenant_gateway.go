package kubeclient

import (
	"kubegems.io/pkg/apis/gems/v1beta1"
)

func (k KubeClient) GetTenantGateway(cluster, name string, _ map[string]string) (*v1beta1.TenantGateway, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1beta1.TenantGateway{}
	err = agentClient.GetObject(&v1beta1.SchemeTenantGateway, obj, nil, &name)
	return obj, err
}

func (k KubeClient) GetTenantGatewayList(cluster string, labelSet map[string]string) (*[]v1beta1.TenantGateway, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1beta1.TenantGateway{}
	err = agentClient.GetObjectList(&v1beta1.SchemeTenantGateway, &list, nil, labelSet)
	return &list, err
}

func (k KubeClient) DeleteTenantGateway(cluster, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(&v1beta1.SchemeTenantGateway, nil, &name)
}

func (k KubeClient) CreateTenantGateway(cluster, name string, data *v1beta1.TenantGateway) (*v1beta1.TenantGateway, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.CreateObject(&v1beta1.SchemeTenantGateway, data, nil, &name)
	return data, err
}

func (k KubeClient) UpdateTenantGateway(cluster, name string, data *v1beta1.TenantGateway) (*v1beta1.TenantGateway, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.UpdateObject(&v1beta1.SchemeTenantGateway, data, nil, &name)
	return data, err
}
