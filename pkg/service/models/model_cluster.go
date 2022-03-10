package models

import (
	"context"
	"errors"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ClusterModeProxy   = "apiServerProxy"
	ClusterModeService = "service"
	ResCluster         = "cluster"
)

// Cluster 集群表
type Cluster struct {
	ID uint `gorm:"primarykey"`
	// 集群名字
	ClusterName string `gorm:"type:varchar(50);uniqueIndex" binding:"required"`
	// APIServer地址 根据kubeconfig添加后，自动填充
	APIServer string `gorm:"type:varchar(250);uniqueIndex"`
	// KubeConfig 配置
	KubeConfig datatypes.JSON `binding:"required"`
	// Vendor 集群提供商(gke tke aliyun selfhosted)
	Vendor string `gorm:"type:varchar(50);default:selfhosted" binding:"required,oneof=selfhosted gke aliyun tke"`
	// ImageRepo 安装kubegems核心组件时使用的镜像仓库
	ImageRepo string `gorm:"type:varchar(255);default:docker.io" binding:"required"`

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

type ClusterInfo struct {
	ID                   uint
	ClusterName          string
	APIServer            string
	Version              string
	AgentAddr            string
	AgentCA              string
	Mode                 string
	Runtime              string
	Environments         []*Environment
	TenantResourceQuotas []*TenantResourceQuota
	ClusterResourceQuota datatypes.JSON
}

type ClusterGetter struct {
	db *gorm.DB
}

func (g ClusterGetter) ListCluster() []string {
	var (
		ret     []string
		cluster Cluster
	)
	g.db.Model(&cluster).Pluck("cluster_name", &ret)
	return ret
}

func (g ClusterGetter) GetManagerCluster(_ context.Context) (string, error) {
	ret := []string{}
	cluster := &Cluster{Primary: true}
	if err := g.db.Where(cluster).Model(cluster).Pluck("cluster_name", &ret).Error; err != nil {
		return "", err
	}
	if len(ret) == 0 {
		return "", errors.New("no manager cluster found")
	}
	return ret[0], nil
}

func (g ClusterGetter) GetByName(name string) (agentAddr, mode string, agentcert, agentkey, agentca, kubeconfig []byte, err error) {
	var cluster Cluster
	if err = g.db.First(&cluster, "cluster_name = ?", name).Error; err != nil {
		return
	}
	agentcert = []byte(cluster.AgentCert)
	agentkey = []byte(cluster.AgentKey)
	agentca = []byte(cluster.AgentCA)
	agentAddr = cluster.AgentAddr
	mode = cluster.Mode
	kubeconfig = []byte(cluster.KubeConfig)
	err = nil
	return
}

type ClusterData struct {
	ID                   uint
	ClusterName          string
	APIServer            string
	Version              string
	AgentAddr            string
	Environments         []*Environment
	TenantResourceQuotas []*TenantResourceQuota
	ClusterResourceQuota datatypes.JSON
}
