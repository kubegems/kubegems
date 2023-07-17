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

package appstore

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/helm"
)

//	@Tags			Appstore
//	@Summary		列出所有的外部应用的charts仓库
//	@Description	列出所有的外部应用的charts仓库
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	handlers.ResponseStruct{Data=handlers.PageData{List=[]models.ChartRepo}}	"repos"
//	@Router			/v1/appstore/repo [get]
//	@Security		JWT
func (h *AppstoreHandler) ListExternalRepo(c *gin.Context) {
	list := []models.ChartRepo{}
	if tx := h.GetDB().WithContext(c.Request.Context()).Find(&list); tx.Error != nil {
		handlers.NotOK(c, tx.Error)
		return
	}
	handlers.OK(c, list)
}

//	@Tags			Appstore
//	@Summary		创建应用商店外部charts仓库
//	@Description	创建应用商店外部charts仓库
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	handlers.ResponseStruct{Data=[]models.ChartRepo}	"repo"
//	@Router			/v1/appstore/repo [post]
//	@Security		JWT
func (h *AppstoreHandler) PutExternalRepo(c *gin.Context) {
	repo := &models.ChartRepo{}
	if err := c.BindJSON(repo); err != nil {
		handlers.NotOK(c, err)
		return
	}

	repository, err := helm.NewLegencyRepository(&helm.RepositoryConfig{Name: repo.ChartRepoName, URL: repo.URL})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	// validate repo
	if _, err := repository.GetIndex(c.Request.Context()); err != nil {
		handlers.NotOK(c, i18n.Errorf(c, "helm chart repo index URL is invalid: %w", err))
		return
	}

	action := i18n.Sprintf(context.TODO(), "create")
	module := i18n.Sprintf(context.TODO(), "external helm chart repo")
	h.SetAuditData(c, action, module, repo.ChartRepoName)
	if err := h.GetDB().WithContext(c.Request.Context()).Save(repo).Error; err != nil {
		handlers.NotOK(c, err)
		return
	} else {
		handlers.OK(c, repo)
		// sync repo
		go func() {
			SyncCharts(context.Background(), repo, helm.RepositoryConfig{URL: h.AppStoreOpt.Addr}, h.GetDB())
		}()
		return
	}
}

//	@Tags			Appstore
//	@Summary		APP 删除外部chart仓库
//	@Description	删除外部chart仓库
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string											true	"repo name"
//	@Success		200		{object}	handlers.ResponseStruct{Data=models.ChartRepo}	"repo"
//	@Router			/v1/appstore/repo/{name} [delete]
//	@Security		JWT
func (h *AppstoreHandler) DeleteExternalRepo(c *gin.Context) {
	action := i18n.Sprintf(context.TODO(), "delete")
	module := i18n.Sprintf(context.TODO(), "external helm chart repo")
	h.SetAuditData(c, action, module, c.Param("name"))
	repo := &models.ChartRepo{ChartRepoName: c.Param("name")}
	if err := h.GetDB().WithContext(c.Request.Context()).Where(repo).Delete(repo).Error; err != nil {
		handlers.NotOK(c, err)
		return
	} else {
		handlers.NoContent(c, repo)
		return
	}
}

//	@Tags			Appstore
//	@Summary		APP 同步外部chart仓库
//	@Description	手动同步外部chart仓库至本地chart museum
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string											true	"repo name"
//	@Success		200		{object}	handlers.ResponseStruct{Data=models.ChartRepo}	"repo"
//	@Router			/v1/appstore/repo/{name}/actions/sync [post]
//	@Security		JWT
func (h *AppstoreHandler) SyncExternalRepo(c *gin.Context) {
	reponame := c.Param("name")
	action := i18n.Sprintf(context.TODO(), "sync")
	module := i18n.Sprintf(context.TODO(), "external helm chart repo")
	h.SetAuditData(c, action, module, reponame)

	if reponame == "" {
		handlers.NotOK(c, errors.New("name required"))
		return
	}
	repo := &models.ChartRepo{ChartRepoName: reponame}
	if err := h.GetDB().WithContext(c.Request.Context()).Where(repo).Take(repo).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	go func() {
		SyncCharts(context.Background(), repo, helm.RepositoryConfig{URL: h.AppStoreOpt.Addr}, h.GetDB())
	}()
	handlers.OK(c, i18n.Sprintf(c, "repo %s started syncing on background", reponame))
}

func SyncCharts(ctx context.Context, repo *models.ChartRepo, localChartMuseum helm.RepositoryConfig, db *gorm.DB) {
	once := sync.Once{}
	onevent := func(e helm.ProcessEvent) {
		// 有事件就更新
		once.Do(func() {
			repo.SyncStatus = models.SyncStatusRunning
			db.Save(repo)
		})
		if e.Error != nil {
			log.Errorf("sync chart repo %s:%s, error: %s", e.Chart.Name, e.Chart.Version, e.Error.Error())
		}
	}

	err := helm.SyncChartsToChartmuseumWithProcess(ctx, helm.RepositoryConfig{
		Name: repo.ChartRepoName,
		URL:  repo.URL,
	}, localChartMuseum, onevent)
	if err != nil {
		if errors.Is(err, helm.ErrSynchronizing) {
			return
		}
		log.Errorf("sync chart repo %s failed: %v", repo.ChartRepoName, err)
		repo.SyncStatus = models.SyncStatusError
		repo.SyncMessage = err.Error()
	} else {
		repo.SyncStatus = models.SyncStatusSuccess
	}
	log.Infof("sync chart repo %s finished", repo.ChartRepoName)
	now := time.Now()
	repo.LastSync = &now
	db.Save(repo)
}
