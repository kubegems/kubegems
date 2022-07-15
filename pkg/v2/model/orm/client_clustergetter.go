// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
