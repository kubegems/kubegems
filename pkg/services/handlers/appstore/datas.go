package appstorehandler

import (
	"time"

	"helm.sh/helm/v3/pkg/chart"
	"kubegems.io/pkg/services/handlers"
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

type AppListInfoResp struct {
	handlers.RespBase
	Data []Chart `json:"data"`
}
