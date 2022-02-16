package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	TenantRoleAdmin    = "admin"
	TenantRoleOrdinary = "ordinary"
	ResTenant          = "tenant"
)

// Tenant 租户表
type Tenant struct {
	ID uint `gorm:"primarykey"`
	// 租户名字
	TenantName string `gorm:"type:varchar(50);uniqueIndex"`
	// 备注
	Remark string
	// 是否激活
	IsActive  bool
	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`

	ResourceQuotas []*TenantResourceQuota
	Users          []*User `gorm:"many2many:tenant_user_rels;"`
	Projects       []*Project
}

// TenantUserRels 租户-用户-关系表
// 租户id-用户id-类型 唯一索引
type TenantUserRels struct {
	ID     uint    `gorm:"primarykey"`
	Tenant *Tenant `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 租户ID
	TenantID uint  `gorm:"uniqueIndex:uniq_idx_tenant_user_rel"`
	User     *User `json:",omitempty" gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 用户ID
	UserID uint `gorm:"uniqueIndex:uniq_idx_tenant_user_rel"`

	// 租户级角色(管理员admin, 普通用户ordinary)
	Role string `gorm:"type:varchar(30)" binding:"required"`
}

type TenantResourceQuota struct {
	ID      uint
	Content datatypes.JSON

	TenantID                   uint                      `gorm:"uniqueIndex:uniq_tenant_cluster" binding:"required"`
	ClusterID                  uint                      `gorm:"uniqueIndex:uniq_tenant_cluster" binding:"required"`
	Tenant                     *Tenant                   `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	Cluster                    *Cluster                  `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	TenantResourceQuotaApply   *TenantResourceQuotaApply `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:SET NULL;"`
	TenantResourceQuotaApplyID *uint
}

const (
	QuotaStatusApproved = "approved"
	QuotaStatusRejected = "rejected"
	QuotaStatusPending  = "pending"
)

// TenantResourceQuotaApply 集群资源申请
type TenantResourceQuotaApply struct {
	ID        uint
	Content   datatypes.JSON
	Status    string    `gorm:"type:varchar(30);"`
	Username  string    `gorm:"type:varchar(255);"`
	UpdatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`
}

// TenantUserSels 租户下用户选项
type TenantUserSels struct {
	ID       uint
	Username string
	Email    string
	Kind     string
}

// TenantSel 租户选项
type TenantSel struct {
	ID         uint
	TenantName string
}

// TenantUserDetail 租户下用户详情
type TenantUserDetail struct {
	ID          uint
	Username    string
	Email       string
	Kind        string
	IsActive    bool
	CreatedAt   time.Time
	LastLoginAt time.Time
}

/*
	删除租户后，需要删除这个租户在各个集群下占用的资源
*/
func (t *Tenant) AfterDelete(tx *gorm.DB) error {
	for _, quota := range t.ResourceQuotas {
		if err := GetKubeClient().DeleteTenant(quota.Cluster.ClusterName, t.TenantName); err != nil {
			return err
		}
	}
	return nil
}

/*
	同步删除对应集群的资源
*/
func (trq *TenantResourceQuota) AfterDelete(tx *gorm.DB) error {
	if err := GetKubeClient().DeleteTenant(trq.Cluster.ClusterName, trq.Tenant.TenantName); err != nil {
		return err
	}
	return nil
}

func (trq *TenantResourceQuota) AfterSave(tx *gorm.DB) error {
	var (
		tenant  Tenant
		cluster Cluster
		rels    []TenantUserRels
	)
	tx.First(&cluster, "id = ?", trq.ClusterID)
	tx.First(&tenant, "id = ?", trq.TenantID)
	tx.Preload("User").Find(&rels, "tenant_id = ?", trq.TenantID)

	admins := []string{}
	members := []string{}
	for _, rel := range rels {
		if rel.Role == TenantRoleAdmin {
			admins = append(admins, rel.User.Username)
		} else {
			members = append(members, rel.User.Username)
		}
	}
	// 创建or更新 租户
	if err := GetKubeClient().CreateOrUpdateTenant(cluster.ClusterName, tenant.TenantName, admins, members); err != nil {
		return err
	}
	// 这儿有个坑，controller还没有成功创建出来TenantResourceQuota，就去更新租户资源，会报错404；先睡会儿把
	<-time.NewTimer(time.Second * 2).C
	// 创建or更新 租户资源
	if err := GetKubeClient().CreateOrUpdateTenantResourceQuota(cluster.ClusterName, tenant.TenantName, trq.Content); err != nil {
		return err
	}
	return nil
}
