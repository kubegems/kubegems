package appstorehandler

import (
	"context"
	"errors"
	"sync"
	"time"

	"kubegems.io/pkg/log"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/utils/helm"
)

const (
	SyncStatusRunning = "running"
	SyncStatusError   = "error"
	SyncStatusSuccess = "success"
)

func SyncCharts(ctx context.Context, repo forms.ChartRepoCommon, localChartMuseum helm.RepositoryConfig, modelClient client.ModelClientIface) {
	once := sync.Once{}
	onevent := func(e helm.ProcessEvent) {
		// 有事件就更新
		once.Do(func() {
			updateObj := forms.ChartRepoCommon{
				SyncStatus: SyncStatusRunning,
			}
			modelClient.Update(ctx, updateObj.Object(), client.WhereNameEqual(repo.Name))
		})
		if e.Error != nil {
			log.Error(e.Error, "sync chart repo failed", "chart", e.Chart.Name, "chart version", e.Chart.Version)
		}
	}

	err := helm.SyncChartsToChartmuseumWithProcess(ctx, helm.RepositoryConfig{
		Name: repo.Name,
		URL:  repo.URL,
	}, localChartMuseum, onevent)

	updateObj := forms.ChartRepoCommon{}
	if err != nil {
		if errors.Is(err, helm.ErrSynchronizing) {
			return
		}
		log.Error(err, "sync repo charts failed", "erpo", repo.Name)
		updateObj.SyncStatus = SyncStatusError
		updateObj.SyncMessage = err.Error()
	} else {
		updateObj.SyncStatus = SyncStatusSuccess
	}
	log.Info("sync repo charts finished", "repo", repo.Name)
	now := time.Now()
	updateObj.LastSync = &now

	modelClient.Update(ctx, updateObj.Object(), client.WhereNameEqual(repo.Name))
}
