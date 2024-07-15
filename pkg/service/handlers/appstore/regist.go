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
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	"kubegems.io/kubegems/pkg/utils/helm"
)

type AppstoreHandler struct {
	base.BaseHandler
	AppStoreOpt       *helm.Options
	ChartmuseumClient *helm.ChartmuseumClient
}

func (h *AppstoreHandler) RegistRouter(rg *gin.RouterGroup) {
	h.ChartmuseumClient = helm.MustNewChartMuseumClient(&helm.RepositoryConfig{URL: h.AppStoreOpt.Addr})

	rg.GET("/appstore/app", h.ListApps)
	rg.GET("/appstore/app/:name", h.AppDetail)
	rg.GET("/appstore/files", h.AppFiles)

	rg.GET("/appstore/repo", h.ListExternalRepo)
	rg.POST("/appstore/repo", h.CheckIsSysADMIN, h.PutExternalRepo)
	rg.DELETE("/appstore/repo/:name", h.CheckIsSysADMIN, h.DeleteExternalRepo)
	rg.POST("/appstore/repo/:name/actions/sync", h.CheckIsSysADMIN, h.SyncExternalRepo)
	rg.POST("/appstore/repo/:name", h.CheckIsSysADMIN, h.UploadCharts)
}
