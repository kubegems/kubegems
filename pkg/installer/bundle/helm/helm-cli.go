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

package helm

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

type ApplyOptions struct {
	DryRun  bool
	Repo    string
	Version string
}

// Download helm chart into cachedir saved as {name}-{version}.tgz file.
func Download(ctx context.Context, repo, name, version, cachedir string) (string, *chart.Chart, error) {
	// check exists
	filename := filepath.Join(cachedir, name+"-"+version+".tgz")
	if _, err := os.Stat(filename); err == nil {
		chart, err := loader.Load(filename)
		if err != nil {
			return filename, nil, err
		}
		return filename, chart, nil
	}
	chartPath, chart, err := LoadAndUpdateChart(ctx, repo, name, version)
	if err != nil {
		return "", nil, err
	}
	intofile, err := filepath.Abs(filepath.Join(cachedir, filepath.Base(chartPath)))
	if err != nil {
		return "", nil, err
	}
	if chartPath == intofile {
		return chartPath, chart, nil
	}
	os.MkdirAll(filepath.Dir(intofile), DefaultDirectoryMode)
	return intofile, chart, os.Rename(chartPath, intofile)
}

// name is the name of the chart
// repo is the url of the chart repository,eg: http://charts.example.com
// if repopath is not empty,download it from repo and set chartNameOrPath to repo/repopath.
// LoadChart loads the chart from the repository
func LoadAndUpdateChart(ctx context.Context, repo, nameOrPath, version string) (string, *chart.Chart, error) {
	chartPath, err := LocateChartSuper(ctx, repo, nameOrPath, version)
	if err != nil {
		return "", nil, err
	}
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", nil, err
	}
	// dependencies update
	if err := action.CheckDependencies(chart, chart.Metadata.Dependencies); err != nil {
		settings := cli.New()
		man := &downloader.Manager{
			Out:              log.Default().Writer(),
			ChartPath:        chartPath,
			SkipUpdate:       false,
			Getters:          getter.All(settings),
			RepositoryConfig: settings.RepositoryConfig,
			RepositoryCache:  settings.RepositoryCache,
			Debug:            settings.Debug,
		}
		if err := man.Update(); err != nil {
			return "", nil, err
		}
		chart, err = loader.Load(chartPath)
		if err != nil {
			return "", nil, err
		}
	}
	return chartPath, chart, nil
}

func LocateChartSuper(ctx context.Context, repoURL, name, version string) (string, error) {
	repou, err := url.Parse(repoURL)
	if err != nil {
		return "", err
	}
	if repou.Scheme != FileProtocolSchema {
		return downloadChart(ctx, repoURL, name, version)
	}
	// handle file:// schema
	index, err := LoadIndex(ctx, repoURL)
	if err != nil {
		return "", err
	}
	cv, err := index.Get(name, version)
	if err != nil {
		return "", err
	}
	if len(cv.URLs) == 0 {
		return "", fmt.Errorf("%v has no downloadable URLs", cv)
	}

	downloadu, err := url.Parse(cv.URLs[0])
	if err != nil {
		return "", fmt.Errorf("parse chart download url: %w", err)
	}

	if !strings.HasSuffix(repou.Path, "/") {
		repou.Path += "/"
	}
	return repou.ResolveReference(downloadu).Path, nil
}

func downloadChart(ctx context.Context, repourl, name, version string) (string, error) {
	settings := cli.New()
	dl := downloader.ChartDownloader{
		Out:              os.Stdout,
		Getters:          getter.All(settings),
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
	}
	if repourl != "" {
		chartURL, err := repo.FindChartInRepoURL(repourl, name, version, "", "", "", dl.Getters)
		if err != nil {
			return "", err
		}
		name = chartURL
	}
	if err := os.MkdirAll(settings.RepositoryCache, DefaultDirectoryMode); err != nil {
		return "", err
	}
	filename, _, err := dl.DownloadTo(name, version, settings.RepositoryCache)
	if err != nil {
		return filename, fmt.Errorf("failed to download %s: %w", name, err)
	}
	return filename, nil
}
