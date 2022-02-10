package models

import (
	"encoding/json"

	"github.com/kubegems/gems/pkg/apis/gems/v1beta1"
	"github.com/kubegems/gems/pkg/controller/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
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

/*
环境的创建，修改，删除，都会触发hook，将状态同步到对应的集群下
*/
func (env *Environment) AfterSave(tx *gorm.DB) error {
	var (
		project       Project
		cluster       Cluster
		spec          v1beta1.EnvironmentSpec
		tmpLimitRange map[string]v1.LimitRangeItem
		limitRange    []v1.LimitRangeItem
		resourceQuota v1.ResourceList
	)
	if e := tx.Preload("Tenant").First(&project, "id = ?", env.ProjectID).Error; e != nil {
		return e
	}
	if e := tx.First(&cluster, "id = ?", env.ClusterID).Error; e != nil {
		return e
	}

	if env.LimitRange != nil {
		e := json.Unmarshal(env.LimitRange, &tmpLimitRange)
		if e != nil {
			return e
		}
	}
	if env.ResourceQuota != nil {
		e := json.Unmarshal(env.ResourceQuota, &resourceQuota)
		if e != nil {
			return e
		}
	}

	for key, v := range tmpLimitRange {
		v.Type = v1.LimitType(key)
		limitRange = append(limitRange, v)
	}
	spec.Namespace = env.Namespace
	spec.Project = project.ProjectName
	spec.Tenant = project.Tenant.TenantName
	spec.LimitRageName = "default"
	spec.ResourceQuotaName = "default"
	spec.DeletePolicy = env.DeletePolicy
	spec.ResourceQuota = resourceQuota
	if len(limitRange) > 0 {
		spec.LimitRage = limitRange
	}
	if e := GetKubeClient().CreateOrUpdateEnvironment(cluster.ClusterName, env.EnvironmentName, spec); e != nil {
		return e
	}
	return nil
}

// 环境删除,同步删除CRD
func (env *Environment) AfterDelete(tx *gorm.DB) error {
	return GetKubeClient().DeleteEnvironment(env.Cluster.ClusterName, env.EnvironmentName)
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
