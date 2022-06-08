package appstorehandler

import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/helm"
	"kubegems.io/kubegems/pkg/v2/models"
)

const (
	SyncStatusRunning = "running"
	SyncStatusError   = "error"
	SyncStatusSuccess = "success"
)

func SyncCharts(ctx context.Context, repo *models.ChartRepo, localChartMuseum helm.RepositoryConfig, db *gorm.DB) {
	once := sync.Once{}
	onevent := func(e helm.ProcessEvent) {
		// 有事件就更新
		once.Do(func() {
			db.Where("id = ?", repo.ID).UpdateColumn("sync_status", SyncStatusRunning)
		})
		if e.Error != nil {
			log.Error(e.Error, "sync chart repo failed", "chart", e.Chart.Name, "chart version", e.Chart.Version)
		}
	}

	err := helm.SyncChartsToChartmuseumWithProcess(ctx, helm.RepositoryConfig{
		Name: repo.Name,
		URL:  repo.URL,
	}, localChartMuseum, onevent)

	updates := map[string]interface{}{}
	if err != nil {
		if errors.Is(err, helm.ErrSynchronizing) {
			return
		}
		log.Error(err, "sync repo charts failed", "erpo", repo.Name)
		updates["sync_status"] = SyncStatusError
		updates["sync_message"] = err.Error()
	} else {
		updates["sync_status"] = SyncStatusSuccess
		updates["last_sync"] = time.Now().Format(time.RFC3339)
	}
	log.Info("sync repo charts finished", "repo", repo.Name)
	db.Where("id = ?", repo.ID).Updates(updates)
}
