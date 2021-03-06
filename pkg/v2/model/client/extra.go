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

package client

import (
	"time"

	v1 "k8s.io/api/core/v1"
	"kubegems.io/kubegems/pkg/apis/gems/v1beta1"
)

type KubeClientIfe interface {
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

type CommonResourceIfe interface {
	GetKind() string
	GetID() uint
	GetTenantID() uint
	GetProjectID() uint
	GetEnvironmentID() uint
	GetVirtualSpaceID() uint
	GetName() string
	GetCluster() string
	GetNamespace() string
	GetOwners() []CommonResourceIfe
}

type CommonUserIfe interface {
	GetID() uint
	GetSystemRoleID() uint
	GetUsername() string
	GetUserKind() string
	GetEmail() string
	GetSource() string
	SetLastLogin(*time.Time)
	UnmarshalBinary(data []byte) error
	MarshalBinary() (data []byte, err error)
}
type UserAuthorityIfe interface {
	// ????????????
	GetResourceRole(kind string, id uint) string
	// ????????????????????????
	IsSystemAdmin() bool
	// ????????????????????????
	IsTenantAdmin(tenantid uint) bool
	// ?????????????????????
	IsTenantMember(tenantid uint) bool
	// ????????????????????????
	IsProjectAdmin(projectid uint) bool
	// ?????????????????????
	IsProjectDev(projectid uint) bool
	// ?????????????????????
	IsProjectTest(projectid uint) bool
	// ?????????????????????
	IsProjectOps(projectid uint) bool
	// ???????????????op
	IsEnvironmentOperator(envid uint) bool
	// ???????????????reader
	IsEnvironmentReader(envid uint) bool
	// ??????????????????????????????
	IsVirtualSpaceAdmin(vsid uint) bool
	// ???????????????????????????
	IsVirtualSpaceMember(vsid uint) bool
	// ?????????????????????????????????
	IsAnyTenantAdmin() bool
}

type ModelCacheIfe interface {
	// ????????????
	BuildCache() error

	// ????????????
	UpsertTenant(tid uint, name string)
	DelTenant(tid uint)
	UpsertProject(tid, pid uint, name string) error
	DelProject(tid, pid uint) error
	UpsertEnvironment(pid, eid uint, name, cluster, namespace string) error
	DelEnvironment(pid, eid uint) error
	UpsertVirtualSpace(vid uint, name string)
	DelVirtualSpace(vid uint)
	FindParents(kind string, id uint) []CommonResourceIfe
	FindResource(kind string, id uint) CommonResourceIfe
	FindEnvironment(cluster, namespace string) CommonResourceIfe

	// ??????????????????
	GetUserAuthority(user CommonUserIfe) UserAuthorityIfe
	FlushUserAuthority(user CommonUserIfe) UserAuthorityIfe

	CacheUserInfo(userinfo CommonUserIfe) error
	GetUserInfo(username string, user CommonUserIfe) error

	// openapi ????????????
	CacheUserInfoViaToken(token string, user CommonUserIfe, ex time.Duration) error
	GetUserInfoViaToken(token string, username string, user CommonUserIfe) error
	UserRequestLimitAllow(user CommonUserIfe, d time.Duration, rate int) (bool, error)
}
