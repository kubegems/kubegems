package appstorehandler

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/emicklei/go-restful/v3"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/services/handlers/base"
	"kubegems.io/pkg/utils/helm"
)

const InternalChartRepoName = "gems"

type Handler struct {
	base.BaseHandler
	AppStoreOpt       *helm.Options
	ChartMuseumClient *helm.ChartmuseumClient
}

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
	RepoURL     string              `json:"repoURL"`
}

type AppFilesResponse struct {
	Files   map[string]string `json:"files" description:"files"`
	App     string            `json:"app" description:"app"`
	Version string            `json:"version" description:"version"`
}

func (h *Handler) ListApps(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	index, err := h.ChartMuseumClient.ListAllChartVersions(ctx, req.PathParameter("repo"))
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	ret := chartVersion2List(index, "")
	handlers.OK(resp, ret)
}

func (h *Handler) RetrieveApp(req *restful.Request, resp *restful.Response) {
	repo := req.PathParameter("repo")
	chart := req.PathParameter("chart")
	ctx := req.Request.Context()
	if repo == "" {
		handlers.NotFound(resp, fmt.Errorf("invalid repo %s", repo))
		return
	}
	var repourl string
	if repo == InternalChartRepoName {
		repourl = "http://gems-chartmuseum.gemcloud-system:8030/gems" // TODO: config
	} else {
		repoData := forms.ChartRepoCommon{}
		if err := h.Model().Get(ctx, repoData.Object(), client.WhereNameEqual(repo)); err != nil {
			handlers.NotFoundOrBadRequest(resp, err)
			return
		}
		repourl = repoData.URL
	}

	index, err := h.ChartMuseumClient.ListChartVersions(ctx, repo, chart)
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	ret := make([]Chart, len(*index))
	for idx, v := range *index {
		ret[idx] = convertChartVersion(v, repourl)
	}
	handlers.OK(resp, ret)
}

func (h *Handler) RetrieveAppFiles(req *restful.Request, resp *restful.Response) {
	repo := req.PathParameter("repo")
	chart := req.PathParameter("chart")
	version := req.PathParameter("version")
	ctx := req.Request.Context()
	chartfiles, err := h.ChartMuseumClient.GetChartBufferedFiles(ctx, repo, chart, version)
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	// convert list to map
	files := map[string]string{}
	for _, v := range chartfiles {
		files[v.Name] = base64.StdEncoding.EncodeToString(v.Data)
	}
	ret := AppFilesResponse{
		Files:   files,
		App:     chart,
		Version: version,
	}
	handlers.OK(resp, ret)
}

func (h *Handler) ListExternalRepos(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ol := forms.ChartRepoCommonList{}
	if err := h.Model().List(ctx, ol.Object()); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, ol.Data())
}

func (h *Handler) CreateExternalRepo(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	repoData := forms.ChartRepoCommon{}
	if err := handlers.BindData(req, repoData); err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	repository, err := helm.NewLegencyRepository(&helm.RepositoryConfig{Name: repoData.Name, URL: repoData.URL})
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	// validate repo
	if _, err := repository.GetIndex(ctx); err != nil {
		handlers.BadRequest(resp, fmt.Errorf("invalid repo index: %w", err))
		return
	}

	if err := h.Model().Create(ctx, repoData.Object()); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, repoData.Data())
	go func() {
		SyncCharts(context.Background(), repoData, helm.RepositoryConfig{URL: h.AppStoreOpt.Addr}, h.Model())
	}()
	return
}

func (h *Handler) DeleteExternalRepo(req *restful.Request, resp *restful.Response) {
	repo := req.PathParameter("repo")
	repoData := forms.ChartRepoCommon{}
	ctx := req.Request.Context()
	if err := h.Model().Delete(ctx, repoData.Object(), client.WhereNameEqual(repo)); err != nil {
		if handlers.IsNotFound(err) {
			handlers.NoContent(resp, nil)
			return
		} else {
			handlers.BadRequest(resp, nil)
			return
		}
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) SyncExternalRepo(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	repo := req.PathParameter("repo")
	repoData := &forms.ChartRepoCommon{}
	if err := h.Model().Get(ctx, repoData.Object(), client.WhereNameEqual(repo)); err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}

	go func() {
		SyncCharts(context.Background(), *repoData.Data(), helm.RepositoryConfig{URL: h.AppStoreOpt.Addr}, h.Model())
	}()
	handlers.OK(resp, fmt.Sprintf("repo %s started sync on background", repo))
}

func chartVersion2List(chartVersionsMap map[string]repo.ChartVersions, repourl string) []Chart {
	ret := []Chart{}
	for _, charts := range chartVersionsMap {
		if len(charts) != 0 {
			ret = append(ret, convertChartVersion(charts[0], repourl))
		}
	}
	return ret
}

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
