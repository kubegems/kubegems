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

package task

import (
	"context"

	clusterhandler "kubegems.io/kubegems/pkg/service/handlers/cluster"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/workflow"
)

type ClusterSyncTasker struct {
	DB *database.Database
	cs *agents.ClientSet
}

func (t *ClusterSyncTasker) Sync(ctx context.Context) error {
	clusters := []*models.Cluster{}
	if err := t.DB.DB().Find(&clusters).Error; err != nil {
		return err
	}
	// in case of cluster upgrade
	return t.cs.ExecuteInEachCluster(context.TODO(), func(ctx context.Context, cli agents.Client) error {
		for _, v := range clusters {
			if v.ClusterName != cli.Name() {
				continue
			}
			if version := cli.Info().APIServerVersion(); v.Version != version {
				if err := t.DB.DB().Model(v).Update("version", version).Error; err != nil {
					return err
				}
			}
			cfg, err := kube.GetKubeRestConfig(v.KubeConfig)
			if err != nil {
				return err
			}
			if exp := clusterhandler.ConfigClientCertExpire(cfg); exp != nil &&
				(v.ClientCertExpireAt == nil || !v.ClientCertExpireAt.Equal(*exp)) {
				if err := t.DB.DB().Model(v).Update("client_cert_expire_at", exp).Error; err != nil {
					return err
				}
			}
			break
		}
		return nil
	})
}

const TaskFunction_ClusterSync = "cluster-sync"

func (t *ClusterSyncTasker) ProvideFuntions() map[string]interface{} {
	return map[string]interface{}{
		TaskFunction_ClusterSync: t.Sync,
	}
}

func (s *ClusterSyncTasker) Crontasks() map[string]Task {
	return map[string]Task{
		"@every 5m": {
			Name:  "cluster-sync",
			Group: "cluster",
			Steps: []workflow.Step{{Function: TaskFunction_ClusterSync}},
		},
	}
}
