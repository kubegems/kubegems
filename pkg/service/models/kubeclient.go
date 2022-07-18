// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"kubegems.io/kubegems/pkg/apis/gems/v1beta1"
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
}
