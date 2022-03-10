package models

import (
	"encoding/json"

	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	"kubegems.io/pkg/utils/resourcequota"
)

/*
ALTER TABLE environments RENAME environment_name TO name
*/

const (
	EnvironmentRoleReader   = "reader"
	EnvironmentRoleOperator = "operator"

	ResEnvironment = "environment"

	EnvironmentMetaTypeDev  = "dev"
	EnvironmentMetaTypeTest = "test"
	EnvironmentMetaTypeProd = "prod"
)

type Environment struct {
	ID        uint   `gorm:"primarykey"`
	Name      string `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_env;index:environment_uniq,unique"`
	Remark    string
	Namespace string `gorm:"type:varchar(50)"`
	// MetaTpe (dev, prod, test, pub ...)
	MetaType       string
	DeletePolicy   string `sql:"DEFAULT:'delNamespace'"`
	Creator        *User
	CreatorID      uint
	Cluster        *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ClusterID      uint
	Project        *Project `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ProjectID      uint     `gorm:"uniqueIndex:uniq_idx_project_env"`
	VirtualSpaceID *uint
	VirtualSpace   *VirtualSpace `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:SET NULL;"`
	ResourceQuota  datatypes.JSON
	LimitRange     datatypes.JSON
	Applications   []*Application `gorm:"many2many:application_environment_rels;"`
	Users          []*User        `gorm:"many2many:environment_user_rels;"`
}

// EnvironmentUserRels
type EnvironmentUserRels struct {
	ID            uint         `gorm:"primarykey"`
	User          *User        `json:",omitempty"`
	Environment   *Environment `json:"omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	UserID        uint         `gorm:"uniqueIndex:uniq_idx_env_user_rel" binding:"required"`
	EnvironmentID uint         `gorm:"uniqueIndex:uniq_idx_env_user_rel" binding:"required"`
	Role          string       `binding:"required,eq=reader|eq=operator"`
}

func FillDefaultLimigrange(env *Environment) []byte {
	defaultLimitRangers := resourcequota.GetDefaultEnvironmentLimitRange()

	kindTmp := map[v1.LimitType]v1.LimitRangeItem{}
	for _, item := range defaultLimitRangers {
		kindTmp[item.Type] = item
	}
	_ = json.Unmarshal(env.LimitRange, &kindTmp)
	ret, _ := json.Marshal(kindTmp)
	return ret
}

type EnvironmentCommon struct {
	ID            uint           `json:"id,omitempty"`
	Name          string         `json:"name,omitempty"`
	Remark        string         `json:"remark,omitempty"`
	Namespace     string         `json:"namespace,omitempty"`
	MetaType      string         `json:"metaType,omitempty"`
	DeletePolicy  string         `json:"deletePolicy,omitempty"`
	Creator       *User          `json:"creator,omitempty"`
	Cluster       *ClusterSimple `json:"cluster,omitempty"`
	ResourceQuota datatypes.JSON `json:"resourceQuota,omitempty"`
	LimitRange    datatypes.JSON `json:"limitRange,omitempty"`
}
