package appstore

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/pagination"
)

const InternalChartRepoName = "gems"

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

// @Tags         Appstore
// @Summary      应用商店-查询所有APP
// @Description  应用商店
// @Accept       json
// @Produce      json
// @Param        page      query     int                                                            false  "page"
// @Param        size      query     int                                                            false  "size"
// @Param        reponame  query     string                                                         false  "reponame"
// @Success      200       {object}  handlers.ResponseStruct{Data=handlers.PageData{List=[]Chart}}  "Apps"
// @Router       /v1/appstore/app [get]
// @Security     JWT
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
	handlers.OK(c, pagination.NewPageDataFromContextReflect(c, ret))
}

// @Tags         Appstore
// @Summary      APP详情
// @Description  查看应用Chart详情
// @Accept       json
// @Produce      json
// @Param        name  path      string                                 true   "name"
// @Param        size  query     string                                 false  "reponame"
// @Success      200   {object}  handlers.ResponseStruct{Data=[]Chart}  "AppDetail"
// @Router       /v1/appstore/app/{name} [get]
// @Security     JWT
func (h *AppstoreHandler) AppDetail(c *gin.Context) {
	reponame := c.Query("reponame")
	name := c.Param("name")

	if reponame == "" {
		reponame = InternalChartRepoName
	}

	repourl := ""
	if reponame == InternalChartRepoName {
		repourl = "http://gems-chartmuseum.gemcloud-system:8030/gems" // TODO: 这里需要统一，由于默认仓库在数据库中无记录。所以之前写死的
	} else {
		modelrepo := &models.ChartRepo{ChartRepoName: reponame}
		if err := h.GetDB().Where(modelrepo).Find(modelrepo).Error; err != nil {
			handlers.NotOK(c, err)
			return
		}
		repourl = modelrepo.URL
	}

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

// @Tags         Appstore
// @Summary      APP Charts文件编码
// @Description  查看App所有Charts文件编码
// @Accept       json
// @Produce      json
// @Param        size     query     string                                          false  "reponame"
// @Param        name     query     string                                          true   "name"
// @Param        version  query     string                                          true   "version"
// @Success      200      {object}  handlers.ResponseStruct{Data=AppFilesResponse}  "AppFiles"
// @Router       /v1/appstore/files [get]
// @Security     JWT
func (h *AppstoreHandler) AppFiles(c *gin.Context) {
	name := c.DefaultQuery("name", "")
	version := c.DefaultQuery("version", "")
	if name == "" || version == "" {
		handlers.NotOK(c, fmt.Errorf("name:%s, version:%s, get name or version failed", name, version))
		return
	}

	reponame := c.Query("reponame")
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
