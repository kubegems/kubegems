package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Cluster 集群表
type Cluster struct {
	ID          uint           `gorm:"primarykey"`
	ClusterName string         `gorm:"type:varchar(50);uniqueIndex" binding:"required"`
	APIServer   string         `gorm:"type:varchar(250);uniqueIndex"` // APIServer地址 根据kubeconfig添加后，自动填充
	KubeConfig  datatypes.JSON `binding:"required"`
	// Vendor 集群提供商(gke tke ack selfhosted)
	Vendor string `gorm:"type:varchar(50);default:selfhosted" binding:"required,oneof=selfhosted gke ack tke"`
	// ImageRepo 安装kubegems核心组件时使用的镜像仓库
	ImageRepo string `gorm:"type:varchar(255);default:docker.io/kubegems" binding:"required"`
	// DefaultStorageClass 默认storageclass, 默认local-path
	DefaultStorageClass  string         `gorm:"type:varchar(255);default:local-path" binding:"required"`
	Version              string         // apiserver version
	AgentAddr            string         // if empty, using apiserver proxy
	AgentCA              string         `json:"-"`
	AgentCert            string         `json:"-"`
	AgentKey             string         `json:"-"`
	Runtime              string         // docker or containerd
	Primary              bool           // 是否主集群
	OversoldConfig       datatypes.JSON // 集群资源超卖设置
	Environments         []*Environment
	TenantResourceQuotas []*TenantResourceQuota
	ClusterResourceQuota datatypes.JSON
	DeletedAt            gorm.DeletedAt // soft delete
}
