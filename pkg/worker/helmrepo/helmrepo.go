package helmrepo

import (
	"context"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers/appstore"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/helm"
)

type Options struct {
	Appstore *helm.Options
	Database *gorm.DB
}

func Run(ctx context.Context, options *Options) error {
	cron := cron.New()
	_, err := cron.AddFunc("@daily", func() {
		syncChartsToChartmuseum(ctx, options)
	})
	if err != nil {
		return err
	}
	go cron.Run()
	<-ctx.Done()
	cron.Stop()
	return nil
}

func syncChartsToChartmuseum(ctx context.Context, options *Options) error {
	repos := []models.ChartRepo{}
	if err := options.Database.Find(&repos).Error; err != nil {
		return err
	}
	for _, repo := range repos {
		log.WithField("repo", repo.ChartRepoName).Info("start sync")
		appstore.SyncCharts(ctx, &repo, helm.RepositoryConfig{URL: options.Appstore.ChartRepoURL}, options.Database)
		log.WithField("repo", repo.ChartRepoName).Info("end sync")
	}
	return nil
}
