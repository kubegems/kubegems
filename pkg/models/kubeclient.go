package models

import (
	"github.com/kubegems/gems/pkg/apis/gems/v1beta1"
	v1 "k8s.io/api/core/v1"
)

var _kubeClient KubeClient

func SetKubeClient(c KubeClient) {
	_kubeClient = c
}

func GetKubeClient() KubeClient {
	return _kubeClient
}

type KubeClient interface {
	GetEnvironment(cluster, name string, _ map[string]string) (*v1beta1.Environment, error)
	PatchEnvironment(cluster, name string, data *v1beta1.Environment) (*v1beta1.Environment, error)
	DeleteEnvironment(clustername, environment string) error
	CreateOrUpdateEnvironment(clustername, environment string, spec v1beta1.EnvironmentSpec) error
	CreateOrUpdateTenant(clustername, tenantname string, admins, members []string) error
	CreateOrUpdateTenantResourceQuota(clustername, tenantname string, content []byte) error
	CreateOrUpdateSecret(clustername, namespace, name string, data map[string][]byte) error
	DeleteSecretIfExist(clustername, namespace, name string) error
	DeleteTenant(clustername, tenantname string) error
	ClusterResourceStatistics(cluster string, ret interface{}) error
	GetServiceAccount(cluster, namespace, name string, labels map[string]string) (*v1.ServiceAccount, error)
	PatchServiceAccount(cluster, namespace, name string, data *v1.ServiceAccount) (*v1.ServiceAccount, error)
}
