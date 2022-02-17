package models

import (
	"encoding/json"

	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	"kubegems.io/pkg/controller/utils"
)

const (
	EnvironmentRoleReader   = "reader"
	EnvironmentRoleOperator = "operator"

	ResEnvironment = "environment"
)

// Environment 环境表
// 环境属于项目，项目id-环境名字 唯一索引
type Environment struct {
	ID uint `gorm:"primarykey"`
	// 环境名字
	EnvironmentName string `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_env;index:environment_uniq,unique"`
	// 环境关联的namespace
	Namespace string `gorm:"type:varchar(50)"`
	// 备注
	Remark string
	// 元类型(开发(dev)，测试(test)，生产(prod))等选项之一
	MetaType string
	// 删除策略(delNamespace删除namespace,delLabels仅删除关联LABEL)
	DeletePolicy string `sql:"DEFAULT:'delNamespace'"`

	// 创建者
	Creator *User
	// 关联的集群
	Cluster *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 所属项目
	Project *Project `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 环境资源限制(这个会和namespace下的ResourceQuota对等)
	ResourceQuota datatypes.JSON
	// 环境下的limitrage
	LimitRange datatypes.JSON
	// 所属项目ID
	ProjectID uint `gorm:"uniqueIndex:uniq_idx_project_env"`
	// 所属集群ID
	ClusterID uint
	// 创建人ID
	CreatorID uint
	// 关联的应用
	Applications []*Application `gorm:"many2many:application_environment_rels;"`
	// 关联的用户
	Users []*User `gorm:"many2many:environment_user_rels;"`

	// 虚拟空间
	VirtualSpaceID *uint
	VirtualSpace   *VirtualSpace `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:SET NULL;"`
}

// EnvironmentUserRels
type EnvironmentUserRels struct {
	ID          uint         `gorm:"primarykey"`
	User        *User        `json:",omitempty"`
	Environment *Environment `json:"omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 用户ID
	UserID uint `gorm:"uniqueIndex:uniq_idx_env_user_rel" binding:"required"`
	// EnvironmentID
	EnvironmentID uint `gorm:"uniqueIndex:uniq_idx_env_user_rel" binding:"required"`

	// 环境级角色("reader", "operator")
	Role string `binding:"required,eq=reader|eq=operator"`
}

func FillDefaultLimigrange(env *Environment) []byte {
	defaultLimitRangers := utils.GetDefaultEnvironmentLimitRange()

	kindTmp := map[v1.LimitType]v1.LimitRangeItem{}
	for _, item := range defaultLimitRangers {
		kindTmp[item.Type] = item
	}
	_ = json.Unmarshal(env.LimitRange, &kindTmp)
	ret, _ := json.Marshal(kindTmp)
	return ret
}
