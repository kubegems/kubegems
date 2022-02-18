package orm

import (
	"reflect"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"kubegems.io/pkg/model/client"
)

var (
	namer       = schema.NamingStrategy{}
	modelFields = sync.Map{}
)

type BaseList struct {
	Page  int64
	Size  int64
	Total int64
}

var models = []interface{}{
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
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(models...)
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

func tableName(objType client.ObjectTypeIface) string {
	// 默认情况下都是类型名字复数
	return *objType.GetKind() + "s"
}

// 关联表的名字
func relTableName(obj1, obj2 client.ObjectTypeIface) string {
	// 默认情况下都是类型名字复数
	return *obj1.GetKind() + "_" + *obj2.GetKind() + "_rels"
}

func ParseFields() {
	for _, ds := range models {
		tablename, fields := parseField(ds)
		modelFields.Store(tablename, fields)
	}
}

func parseField(obj interface{}) (string, []string) {
	o := obj.(client.Object)
	tableName := *o.GetKind()
	rt := reflect.TypeOf(obj)
	rt.Elem().NumField()
	fields := []string{}
	for idx := 0; idx < rt.Elem().NumField(); idx++ {
		field := rt.Elem().Field(idx)
		fields = append(fields, namer.ColumnName(tableName, field.Name))
	}
	return tableName, fields
}
