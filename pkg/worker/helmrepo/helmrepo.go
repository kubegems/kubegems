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

package helmrepo

import (
	"context"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers/appstore"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/helm"
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
		appstore.SyncCharts(ctx, &repo, helm.RepositoryConfig{URL: options.Appstore.Addr}, options.Database)
		log.WithField("repo", repo.ChartRepoName).Info("end sync")
	}
	return nil
}
