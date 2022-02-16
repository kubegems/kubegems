package appstore

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
	"kubegems.io/pkg/utils/helm"
)

type AppstoreHandler struct {
	base.BaseHandler
	AppStoreOpt       *helm.Options
	ChartmuseumClient *helm.ChartmuseumClient
}

func (h *AppstoreHandler) RegistRouter(rg *gin.RouterGroup) {
	h.ChartmuseumClient = helm.MustNewChartMuseumClient(&helm.RepositoryConfig{URL: h.AppStoreOpt.ChartRepoUrl})

	rg.GET("/appstore/app", h.ListApps)
	rg.GET("/appstore/app/:name", h.AppDetail)
	rg.GET("/appstore/files", h.AppFiles)

	rg.GET("/appstore/repo", h.ListExternalRepo)
	rg.POST("/appstore/repo", h.CheckIsSysADMIN, h.PutExternalRepo)
	rg.DELETE("/appstore/repo/:name", h.CheckIsSysADMIN, h.DeleteExternalRepo)
	rg.POST("/appstore/repo/:name/actions/sync", h.CheckIsSysADMIN, h.SyncExternalRepo)
}
