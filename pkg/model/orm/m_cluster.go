package orm

import "gorm.io/datatypes"

// Cluster 集群表
// +gen type:object kind:cluster pkcolume:id pkfield:ID preloads:TenantResourceQuotas
type Cluster struct {
	ID uint `gorm:"primarykey"`
	// 集群名字
	ClusterName string `gorm:"type:varchar(50);uniqueIndex" binding:"required"`
	// APIServer地址 根据kubeconfig添加后，自动填充
	APIServer string `gorm:"type:varchar(250);uniqueIndex"`
	// KubeConfig 配置
	KubeConfig datatypes.JSON `binding:"required"`

	// Version 版本
	Version string
	// Agent地址
	AgentAddr string
	// AgentCA
	AgentCA string `json:"-"`
	// Agent证书
	AgentCert string `json:"-"`
	// Agentkey
	AgentKey string `json:"-"`

	Mode string `json:"-"`

	Runtime string // docker or containerd
	// 是否主集群
	Primary bool

	// 集群资源超卖设置
	OversoldConfig datatypes.JSON

	Environments         []*Environment
	TenantResourceQuotas []*TenantResourceQuota
	ClusterResourceQuota datatypes.JSON
}
