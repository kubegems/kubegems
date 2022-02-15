package orm

import (
	"gorm.io/gorm"
	"kubegems.io/pkg/model/client"
)

type BaseList struct {
	Page  int64
	Size  int64
	Total int64
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// 审计表
		&AuditLog{},
		// 用户表
		&User{},
		// 系统角色表
		&SystemRole{},
		// 租户表
		&Tenant{},
		// 租户成员关系表
		&TenantUserRel{},
		// 租户集群资源表
		&TenantResourceQuota{},
		// 项目表
		&Project{},
		// 项目成员关系表
		&ProjectUserRel{},
		// 环境表
		&Environment{},
		// 环境成员关系表
		&EnvironmentUserRel{},
		// 应用表
		&Application{},
		// 集群表
		&Cluster{},
		// 镜像仓库表
		&Registry{},
		// 日志查询历史表
		&LogQueryHistory{},
		// 日志查询快照表
		&LogQuerySnapshot{},
		// workload 资源建议表
		&Workload{},
		// 容器资源建议表
		&Container{},
		// 租户集群资源申请表
		&TenantResourceQuotaApply{},
		// ??
		&EnvironmentResource{},
		// 消息表
		&Message{},
		// 用户消息表
		&UserMessageStatus{},
		// helmChart仓库
		&ChartRepo{},
		// 虚拟空间表
		&VirtualSpace{},
		// 虚拟空间用户表
		&VirtualSpaceUserRel{},
		// 虚拟域名表
		&VirtualDomain{},
	)
}

func kubeClient(tx *gorm.DB) client.KubeClientIfe {
	plugin := tx.Config.Plugins["kubeclient"]
	kubeclient := plugin.(client.KubeClientIfe)
	return kubeclient
}

func contains(arr []string, t string) bool {
	for _, ar := range arr {
		if ar == t {
			return true
		}
	}
	return false
}

func tableName(objType client.ObjectTypeIfe) string {
	// 默认情况下都是类型名字复数
	return *objType.GetKind() + "s"
}

func objIDFiled(objType client.ObjectTypeIfe) string {
	return *objType.GetKind() + "_id"
}
