package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ProjectRoleAdmin = "admin"
	ProjectRoleDev   = "dev"
	ProjectRoleTest  = "test"
	ProjectRoleOps   = "ops"

	ResProject = "project"
)

// Project 项目表
type Project struct {
	ID uint `gorm:"primarykey"`
	// 创建时间
	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`
	// 项目名字
	ProjectName string `gorm:"type:varchar(50);uniqueIndex:uniq_idx_tenant_project_name"`
	// 项目别名
	ProjectAlias string `gorm:"type:varchar(50)"`
	// 项目备注
	Remark string
	// 项目资源限制
	ResourceQuota datatypes.JSON

	Applications []*Application
	Environments []*Environment
	Registries   []*Registry
	Users        []*User `gorm:"many2many:project_user_rels;"`
	Tenant       *Tenant `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 所属的租户ID
	TenantID uint `gorm:"uniqueIndex:uniq_idx_tenant_project_name"`
}

// ProjectUserRels
type ProjectUserRels struct {
	ID      uint     `gorm:"primarykey"`
	User    *User    `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Project *Project `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 用户ID
	UserID uint `gorm:"uniqueIndex:uniq_idx_project_user_rel" binding:"required"`
	// ProjectID
	ProjectID uint `gorm:"uniqueIndex:uniq_idx_project_user_rel" binding:"required"`

	// 项目级角色(管理员admin, 开发dev, 测试test, 运维ops)
	Role string `gorm:"type:varchar(30)" binding:"required,eq=admin|eq=test|eq=dev|eq=ops"`
}

/*
	删除项目后
	删除各个集群的环境(tenv),tenv本身删除是Controller自带垃圾回收的，其ns下所有资源将清空
*/
func (p *Project) AfterDelete(tx *gorm.DB) error {
	for _, env := range p.Environments {
		e := GetKubeClient().DeleteEnvironment(env.Cluster.ClusterName, env.EnvironmentName)
		if e != nil {
			return e
		}
	}
	// TODO: 删除 GIT 中的数据
	// TODO: 删除 ARGO 中的数据
	return nil
}
