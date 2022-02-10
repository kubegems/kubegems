package appstore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/service/handlers"
	"github.com/kubegems/gems/pkg/utils/helm"
	"gorm.io/gorm"
)

// @Tags Appstore
// @Summary APP 获取外部chart仓库
// @Description  获取外部chart仓库
// @Accept json
// @Produce json
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.ChartRepo}} "repos"
// @Router /v1/appstore/repo [get]
// @Security JWT
func (h *AppstoreHandler) ListExternalRepo(c *gin.Context) {
	list := []models.ChartRepo{}
	if tx := h.GetDB().Find(&list); tx.Error != nil {
		handlers.NotOK(c, tx.Error)
		return
	}
	handlers.OK(c, list)
}

// @Tags Appstore
// @Summary APP 创建外部chart仓库
// @Description  创建外部chart仓库
// @Accept json
// @Produce json
// @Success 200 {object} handlers.ResponseStruct{Data=[]models.ChartRepo} "repo"
// @Router /v1/appstore/repo [post]
// @Security JWT
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
		handlers.NotOK(c, fmt.Errorf("invalid repo index: %w", err))
		return
	}

	h.SetAuditData(c, "创建", "外部charts库", repo.ChartRepoName)
	if err := h.GetDB().Save(repo).Error; err != nil {
		handlers.NotOK(c, err)
		return
	} else {
		handlers.OK(c, repo)
		// sync repo
		go func() {
			SyncCharts(context.Background(), repo, helm.RepositoryConfig{URL: h.AppStoreOpt.ChartRepoUrl}, h.GetDB())
		}()
		return
	}
}

// @Tags Appstore
// @Summary APP 删除外部chart仓库
// @Description  删除外部chart仓库
// @Accept json
// @Produce json
// @Param name query string true "repo name"
// @Success 200 {object} handlers.ResponseStruct{Data=models.ChartRepo} "repo"
// @Router /v1/appstore/repo/{name} [delete]
// @Security JWT
func (h *AppstoreHandler) DeleteExternalRepo(c *gin.Context) {
	h.SetAuditData(c, "删除", "外部charts库", c.Param("name"))
	repo := &models.ChartRepo{ChartRepoName: c.Param("name")}
	if err := h.GetDB().Where(repo).Delete(repo).Error; err != nil {
		handlers.NotOK(c, err)
		return
	} else {
		handlers.NoContent(c, repo)
		return
	}
}

// @Tags Appstore
// @Summary APP 同步外部chart仓库
// @Description  手动同步外部chart仓库至本地chart museum
// @Accept json
// @Produce json
// @Param name query string true "repo name"
// @Success 200 {object} handlers.ResponseStruct{Data=models.ChartRepo} "repo"
// @Router /v1/appstore/repo/{name}/actions/sync [post]
// @Security JWT
func (h *AppstoreHandler) SyncExternalRepo(c *gin.Context) {
	reponame := c.Param("name")
	h.SetAuditData(c, "同步", "外部charts库", c.Param("name"))

	if reponame == "" {
		handlers.NotOK(c, errors.New("name required"))
		return
	}
	repo := &models.ChartRepo{ChartRepoName: reponame}
	if err := h.GetDB().Where(repo).Take(repo).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	go func() {
		SyncCharts(context.Background(), repo, helm.RepositoryConfig{URL: h.AppStoreOpt.ChartRepoUrl}, h.GetDB())
	}()
	handlers.OK(c, fmt.Sprintf("repo %s started sync on background", reponame))
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
			log.Errorf("sync chart %s:%s, error: %s", e.Chart.Name, e.Chart.Version, e.Error.Error())
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
		log.Errorf("sync repo %s charts failed: %v", repo.ChartRepoName, err)
		repo.SyncStatus = models.SyncStatusError
		repo.SyncMessage = err.Error()
	} else {
		repo.SyncStatus = models.SyncStatusSuccess
	}
	log.Infof("sync repo %s charts finished", repo.ChartRepoName)
	now := time.Now()
	repo.LastSync = &now
	db.Save(repo)
}
