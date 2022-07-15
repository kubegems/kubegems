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

	ClusterTableName = "clusters"
)

/*
ALTER TABLE clusters RENAME cluster_name TO name
*/

// Cluster 集群表
type Cluster struct {
	ID                   uint           `gorm:"primarykey"`
	Name                 string         `gorm:"type:varchar(50);uniqueIndex" binding:"required"`
	APIServer            string         `gorm:"type:varchar(250);uniqueIndex"`
	KubeConfig           datatypes.JSON `binding:"required"`
	Version              string
	AgentAddr            string
	AgentCA              string `json:"-"`
	AgentCert            string `json:"-"`
	AgentKey             string `json:"-"`
	Mode                 string `json:"-"`
	Runtime              string // docker or containerd
	Primary              bool
	OversoldConfig       datatypes.JSON
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

type ClusterSimple struct {
	ID        uint
	Name      string
	APIServer string
	Version   string
}

func (ClusterSimple) TableName() string {
	return ClusterTableName
}
