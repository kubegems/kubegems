package appstorehandler

import (
	"time"

	"helm.sh/helm/v3/pkg/chart"
	"kubegems.io/kubegems/pkg/v2/models"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
)

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

type AppFiles struct {
	Files   map[string]string `json:"files" description:"files"`
	App     string            `json:"app" description:"app"`
	Version string            `json:"version" description:"version"`
}

type AppListInfoResp struct {
	handlers.RespBase
	Data []Chart `json:"data"`
}

type AppFilesResp struct {
	handlers.RespBase
	Data AppFiles `json:"data"`
}

type ChartRepoListResp struct {
	handlers.ListBase
	Data []models.ChartRepo `json:"list"`
}

type ChartRepoResp struct {
	handlers.RespBase
	Data []models.ChartRepo `json:"data"`
}
