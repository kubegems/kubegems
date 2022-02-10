package task

import (
	"context"

	"github.com/kubegems/gems/pkg/handlers/appstore"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/utils/database"
	"github.com/kubegems/gems/pkg/utils/helm"
	"github.com/kubegems/gems/pkg/utils/workflow"
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
