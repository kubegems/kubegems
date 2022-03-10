package orm

import (
	"context"
	"errors"
)

/*
ListCluster
GetByName
GetManagerCluster
实现ClusterGetter接口
*/

func (c *Client) ListCluster() []string {
	var clusters []string
	c.db.Table(tableName(&Cluster{})).Pluck("cluster_name", &clusters)
	return clusters
}

func (c *Client) GetByName(name string) (agentAddr, mode string, agentcert, agentkey, agentca, kubeconfig []byte, err error) {
	var cluster Cluster
	if err = c.db.First(&cluster, "cluster_name = ?", name).Error; err != nil {
		return
	}
	agentcert = []byte(cluster.AgentCert)
	agentkey = []byte(cluster.AgentKey)
	agentca = []byte(cluster.AgentCA)
	agentAddr = cluster.AgentAddr
	mode = cluster.Mode
	kubeconfig = []byte(cluster.KubeConfig)
	return
}

func (c *Client) GetManagerCluster(ctx context.Context) (string, error) {
	ret := []string{}
	cluster := &Cluster{Primary: true}
	if err := c.db.Where(cluster).Model(cluster).Pluck("cluster_name", &ret).Error; err != nil {
		return "", err
	}
	if len(ret) == 0 {
		return "", errors.New("no manager cluster found")
	}
	return ret[0], nil
}
