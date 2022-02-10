package helmrepo

import (
	"context"

	"github.com/kubegems/gems/pkg/handlers/appstore"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/utils/chartmuseum"
	"github.com/kubegems/gems/pkg/utils/helm"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type Options struct {
	Appstore *chartmuseum.AppstoreOptions
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
		appstore.SyncCharts(ctx, &repo, helm.RepositoryConfig{URL: options.Appstore.ChartRepoUrl}, options.Database)
		log.WithField("repo", repo.ChartRepoName).Info("end sync")
	}
	return nil
}
