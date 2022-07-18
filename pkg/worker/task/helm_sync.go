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

	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers/appstore"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/helm"
	"kubegems.io/kubegems/pkg/utils/workflow"
)

type HelmSyncTasker struct {
	DB           *database.Database
	ChartRepoUrl string
}

func (t *HelmSyncTasker) SyncCharts(ctx context.Context) error {
	repos := []models.ChartRepo{}
	if err := t.DB.DB().Find(&repos).Error; err != nil {
		return err
	}
	for _, repo := range repos {
		log.WithField("repo", repo.ChartRepoName).Info("start sync")
		appstore.SyncCharts(ctx, &repo, helm.RepositoryConfig{URL: t.ChartRepoUrl}, t.DB.DB())
		log.WithField("repo", repo.ChartRepoName).Info("end sync")
	}
	return nil
}

const TaskFunction_HelmSyncCharts = "helm-sync-charts"

func (t *HelmSyncTasker) ProvideFuntions() map[string]interface{} {
	return map[string]interface{}{
		TaskFunction_HelmSyncCharts: t.SyncCharts,
	}
}

func (s *HelmSyncTasker) Crontasks() map[string]Task {
	return map[string]Task{
		"@daily": {
			Name:  "daily-sync",
			Group: "helm",
			Steps: []workflow.Step{{Function: TaskFunction_HelmSyncCharts}},
		},
	}
}
