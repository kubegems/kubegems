package kubeclient

import (
	"encoding/json"

	"github.com/kubegems/gems/pkg/apis/gems/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func (k KubeClient) GetTenant(cluster, name string, _ map[string]string) (*v1beta1.Tenant, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1beta1.Tenant{}
	err = agentClient.GetObject(&v1beta1.SchemeTenant, obj, nil, &name)
	return obj, err
}

func (k KubeClient) GetTenantList(cluster string, labelSet map[string]string) (*[]v1beta1.Tenant, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1beta1.Tenant{}
	err = agentClient.GetObjectList(&v1beta1.SchemeTenant, &list, nil, labelSet)
	return &list, err
}

func (k KubeClient) DeleteTenant(cluster, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(&v1beta1.SchemeTenant, nil, &name)
}

func (k KubeClient) CreateTenant(cluster, name string, data *v1beta1.Tenant) (*v1beta1.Tenant, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.CreateObject(&v1beta1.SchemeTenant, data, nil, &name)
	return data, err
}

func (k KubeClient) PatchTenant(cluster, name string, data *v1beta1.Tenant) (*v1beta1.Tenant, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.PatchObject(&v1beta1.SchemeTenant, data, nil, &name)
	return data, err
}

func (k KubeClient) UpdateTenant(cluster, name string, data *v1beta1.Tenant) (*v1beta1.Tenant, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.UpdateObject(&v1beta1.SchemeTenant, data, nil, &name)
	return data, err
}

func (k KubeClient) CreateOrUpdateTenant(clustername, tenantname string, admins, members []string) error {
	existTenant, err := GetClient().GetTenant(clustername, tenantname, nil)
	if existTenant == nil || err != nil {
		crdTenant := &v1beta1.Tenant{
			Spec: v1beta1.TenantSpec{
				TenantName: tenantname,
				Admin:      admins,
				Members:    members,
			},
		}
		crdTenant.Name = tenantname
		_, err := GetClient().CreateTenant(clustername, tenantname, crdTenant)
		if err != nil {
			return err
		}
	} else {
		existTenant.Spec = v1beta1.TenantSpec{
			TenantName: tenantname,
			Admin:      admins,
			Members:    members,
		}
		_, err := GetClient().UpdateTenant(clustername, tenantname, existTenant)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k KubeClient) CreateOrUpdateTenantResourceQuota(clustername, tenantname string, data []byte) error {
	var hard corev1.ResourceList
	if err := json.Unmarshal(data, &hard); err != nil {
		return err
	}
	tquota, _ := GetClient().GetTenantResourceQuota(clustername, tenantname, nil)
	if tquota == nil {
		tmp := &v1beta1.TenantResourceQuota{
			Spec: v1beta1.TenantResourceQuotaSpec{
				Hard: hard,
			},
		}
		tmp.Name = tenantname
		if _, err := GetClient().CreateTenantResourceQuota(clustername, tenantname, tmp); err != nil {
			return err
		}
		return nil
	}
	tquota.Spec.Hard = hard
	if _, err := GetClient().PatchTenantResourceQuota(clustername, tenantname, tquota); err != nil {
		return err
	}
	return nil
}
