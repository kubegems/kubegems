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
	"log"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
)

type ApplyOptions struct {
	DryRun  bool
	Repo    string
	Version string
}

// name is the name of the chart
// repo is the url of the chart repository,eg: http://charts.example.com
// if repopath is not empty,download it from repo and set chartNameOrPath to repo/repopath.
// LoadChart loads the chart from the repository
func LoadAndUpdateChart(ctx context.Context, repo, nameOrPath, version string) (string, *chart.Chart, error) {
	chartPathOptions := action.ChartPathOptions{RepoURL: repo, Version: version}
	settings := cli.New()
	chartPath, err := chartPathOptions.LocateChart(nameOrPath, settings)
	if err != nil {
		return "", nil, err
	}
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", nil, err
	}
	// dependencies update
	if err := action.CheckDependencies(chart, chart.Metadata.Dependencies); err != nil {
		man := &downloader.Manager{
			Out:              log.Default().Writer(),
			ChartPath:        chartPath,
			Keyring:          chartPathOptions.Keyring,
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

func Download(ctx context.Context, repo, name, version, cachedir string) (*chart.Chart, error) {
	// check exists
	filename := filepath.Join(cachedir, name+"-"+version+".tgz")
	if _, err := os.Stat(filename); err == nil {
		return loader.Load(filename)
	}
	chartPath, chart, err := LoadAndUpdateChart(ctx, repo, name, version)
	if err != nil {
		return nil, err
	}
	os.MkdirAll(cachedir, DefaultDirectoryMode)
	intofile := filepath.Join(cachedir, filepath.Base(chartPath))
	return chart, os.Rename(chartPath, intofile)
}
