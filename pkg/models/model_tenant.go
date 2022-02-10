package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubegems/gems/pkg/datas"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
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

type OversoldConfig struct {
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Storage float64 `json:"storage"`
}

func ValidateTenantResourceQuota(clusterOversold []byte, clustername string, origin, need v1.ResourceList) error {
	statistics := &datas.ClusterResourceStatistics{}
	if err := GetKubeClient().ClusterResourceStatistics(clustername, statistics); err != nil {
		return fmt.Errorf("验证资源错误 %v", err)
	}

	// 当前的限制
	originCPU := origin[v1.ResourceLimitsCPU]
	originMemory := origin[v1.ResourceLimitsMemory]
	originStorage := origin[v1.ResourceRequestsStorage]

	// 当前申请的资源量
	needCPU := need[v1.ResourceLimitsCPU]
	needMemory := need[v1.ResourceLimitsMemory]
	needStorage := need[v1.ResourceRequestsStorage]

	needCpuV := needCPU.Value() - originCPU.Value()
	needMemoryV := needMemory.Value() - originMemory.Value()
	needStorageV := needStorage.Value() - originStorage.Value()

	// 资源超分验证
	oversoldConfig := OversoldConfig{
		CPU:     1,
		Memory:  1,
		Storage: 1,
	}
	if len(clusterOversold) == 0 {
		clusterOversold = []byte("{}")
	}
	err := json.Unmarshal(clusterOversold, &oversoldConfig)
	if err != nil {
		return fmt.Errorf("资源超卖配置错误 %w", err)
	}

	// 已经分配了的
	allocatedCpu := statistics.TenantAllocated[v1.ResourceLimitsCPU]
	allocatedMemory := statistics.TenantAllocated[v1.ResourceLimitsMemory]
	allocatedStorage := statistics.TenantAllocated[v1.ResourceRequestsStorage]

	// 实际总量
	capacityCpu := statistics.Capacity[v1.ResourceCPU]
	capacityMemory := statistics.Capacity[v1.ResourceMemory]
	capacityStorage := statistics.Capacity[v1.ResourceEphemeralStorage]

	// 超分总量
	oCapacityCPU := int64(float64(capacityCpu.AsDec().UnscaledBig().Int64()) * oversoldConfig.CPU)
	oCapacityMemory := int64(float64(capacityMemory.AsDec().UnscaledBig().Int64()) * oversoldConfig.Memory)
	oCapacityStorage := int64(float64(capacityStorage.AsDec().UnscaledBig().Int64()) * oversoldConfig.Storage)

	leftCpu := oCapacityCPU - allocatedCpu.AsDec().UnscaledBig().Int64()
	leftMemory := oCapacityMemory - allocatedMemory.AsDec().UnscaledBig().Int64()
	leftStorage := oCapacityStorage - allocatedStorage.AsDec().UnscaledBig().Int64()

	errRes := []string{}
	if leftCpu < needCpuV {
		errRes = append(errRes, fmt.Sprintf("CPU(left %v but need %v)", leftCpu, needCpuV))
	}
	if leftMemory < needMemoryV {
		errRes = append(errRes, fmt.Sprintf("Memory(left %v but need %v)", leftMemory, needMemoryV))
	}
	if leftStorage < needStorageV {
		errRes = append(errRes, fmt.Sprintf("Storage(left %v but need %v)", leftStorage, needStorageV))
	}
	if len(errRes) > 0 {
		return fmt.Errorf("集群资源不足%v", errRes)
	}
	return nil
}
