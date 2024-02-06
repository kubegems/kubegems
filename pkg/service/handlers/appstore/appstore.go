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
	"encoding/base64"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/library/rest/response"
)

const InternalChartRepoName = "kubegems"

type Chart struct {
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Description string              `json:"description"`
	ApiVersion  string              `json:"apiVersion"`
	AppVersion  string              `json:"appVersion"`
	Created     time.Time           `json:"created"`
	Digest      string              `json:"digest"`
	Urls        []string            `json:"urls"`
	Keywords    []string            `json:"keywords"`
	Maintainers []*chart.Maintainer `json:"maintainers"`
	Tags        string              `json:"tags"`
	RepoURL     string              `json:"repoURL"` // 仓库地址
}

func (c Chart) GetName() string {
	return c.Name
}

func (c Chart) GetCreationTimestamp() metav1.Time {
	return metav1.Time{Time: c.Created}
}

// @Tags			Appstore
// @Summary		应用商店-查询所有APP
// @Description	应用商店
// @Accept			json
// @Produce		json
// @Param			page		query		int																false	"page"
// @Param			size		query		int																false	"size"
// @Param			reponame	query		string															false	"reponame"
// @Success		200			{object}	handlers.ResponseStruct{Data=handlers.PageData{List=[]Chart}}	"Apps"
// @Router			/v1/appstore/app [get]
// @Security		JWT
func (h *AppstoreHandler) ListApps(c *gin.Context) {
	reponame := c.Query("reponame")
	if reponame == "" {
		reponame = InternalChartRepoName
	}

	index, err := h.ChartmuseumClient.ListAllChartVersions(c.Request.Context(), reponame)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	ret := []Chart{}
	for _, v := range index {
		if len(v) == 0 {
			continue
		}
		// 取最新版本
		ret = append(ret, convertChartVersion(v[0], ""))
	}
	pagedata := response.PageObjectFromRequest(c.Request, ret)
	handlers.OK(c, pagedata)
}

// @Tags			Appstore
// @Summary		APP详情
// @Description	查看应用Chart详情
// @Accept			json
// @Produce		json
// @Param			name	path		string									true	"name"
// @Param			size	query		string									false	"reponame"
// @Success		200		{object}	handlers.ResponseStruct{Data=[]Chart}	"AppDetail"
// @Router			/v1/appstore/app/{name} [get]
// @Security		JWT
func (h *AppstoreHandler) AppDetail(c *gin.Context) {
	reponame := c.Query("reponame")
	name := c.Param("name")

	if reponame == "" {
		reponame = InternalChartRepoName
	}

	repourl := strings.TrimSuffix(h.AppStoreOpt.Addr, "/") + "/" + reponame

	index, err := h.ChartmuseumClient.ListChartVersions(c.Request.Context(), reponame, name)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	// 针对前端显示屏蔽部分字段
	ret := make([]Chart, len(*index))
	for i, v := range *index {
		ret[i] = convertChartVersion(v, repourl)
	}
	handlers.OK(c, ret)
}

type AppFilesResponse struct {
	Files   map[string]string `json:"files" description:"files"`
	App     string            `json:"app" description:"app"`
	Version string            `json:"version" description:"version"`
}

// @Tags			Appstore
// @Summary		APP Charts文件编码
// @Description	查看App所有Charts文件编码
// @Accept			json
// @Produce		json
// @Param			size	query		string											false	"reponame"
// @Param			name	query		string											true	"name"
// @Param			version	query		string											true	"version"
// @Success		200		{object}	handlers.ResponseStruct{Data=AppFilesResponse}	"AppFiles"
// @Router			/v1/appstore/files [get]
// @Security		JWT
func (h *AppstoreHandler) AppFiles(c *gin.Context) {
	name := c.DefaultQuery("name", "")
	version := c.DefaultQuery("version", "")
	reponame := c.Query("reponame")
	if name == "" || version == "" {
		handlers.NotOK(c, i18n.Errorf(c, "invalid parameters: name=%s, version=%s", name, version))
		return
	}

	if reponame == "" {
		reponame = InternalChartRepoName
	}
	ctx := c.Request.Context()
	chartfiles, err := h.ChartmuseumClient.GetChartBufferedFiles(ctx, reponame, name, version)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	// convert list to map
	files := map[string]string{}
	for _, v := range chartfiles {
		files[v.Name] = base64.StdEncoding.EncodeToString(v.Data)
	}
	ret := AppFilesResponse{
		Files:   files,
		App:     name,
		Version: version,
	}
	handlers.OK(c, ret)
}

// 针对前端显示屏蔽部分字段
func convertChartVersion(cv *repo.ChartVersion, repourl string) Chart {
	return Chart{
		Name:        cv.Name,
		Version:     cv.Version,
		Description: cv.Description,
		ApiVersion:  cv.APIVersion,
		AppVersion:  cv.AppVersion,
		Created:     cv.Created,
		Digest:      cv.Digest,
		Urls:        cv.URLs,
		Keywords:    cv.Keywords,
		Maintainers: cv.Maintainers,
		Tags:        cv.Tags,
		RepoURL:     repourl,
	}
}
