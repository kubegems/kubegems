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
	// 资源角色
	GetResourceRole(kind string, id uint) string
	// 是否是系统管理员
	IsSystemAdmin() bool
	// 是否是租户管理员
	IsTenantAdmin(tenantid uint) bool
	// 是否是租户成员
	IsTenantMember(tenantid uint) bool
	// 是否是项目管理员
	IsProjectAdmin(projectid uint) bool
	// 是否是项目开发
	IsProjectDev(projectid uint) bool
	// 是否是项目测试
	IsProjectTest(projectid uint) bool
	// 是否是项目运维
	IsProjectOps(projectid uint) bool
	// 是否是环境op
	IsEnvironmentOperator(envid uint) bool
	// 是否是环境reader
	IsEnvironmentReader(envid uint) bool
	// 是否是虚拟空间管理员
	IsVirtualSpaceAdmin(vsid uint) bool
	// 是否是虚拟空间成员
	IsVirtualSpaceMember(vsid uint) bool
	// 是否是一个租户的管理员
	IsAnyTenantAdmin() bool
}

type ModelCacheIfe interface {
	// 构建缓存
	BuildCache() error

	// 资源相关
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

	// 普通用户相关
	GetUserAuthority(user CommonUserIfe) UserAuthorityIfe
	FlushUserAuthority(user CommonUserIfe) UserAuthorityIfe

	CacheUserInfo(userinfo CommonUserIfe) error
	GetUserInfo(username string, user CommonUserIfe) error

	// openapi 相关用户
	CacheUserInfoViaToken(token string, user CommonUserIfe, ex time.Duration) error
	GetUserInfoViaToken(token string, username string, user CommonUserIfe) error
	UserRequestLimitAllow(user CommonUserIfe, d time.Duration, rate int) (bool, error)
}
