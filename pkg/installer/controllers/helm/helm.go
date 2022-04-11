package helm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/client-go/rest"
)

type Helm struct {
	Config *rest.Config
}

type RemoveOptions struct {
	DryRun bool
}

func (h *Helm) RemoveChart(ctx context.Context, releaseName, releaseNamespace string, options RemoveOptions) (*release.Release, error) {
	log := logr.FromContextOrDiscard(ctx)
	cfg, err := NewHelmConfig(releaseNamespace, h.Config)
	if err != nil {
		return nil, err
	}
	exist, err := action.NewGet(cfg).Run(releaseName)
	if err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, err
		}
		return nil, nil
	}
	log.Info("uninstalling")
	uninstall := action.NewUninstall(cfg)
	uninstalledRelease, err := uninstall.Run(exist.Name)
	if err != nil {
		return nil, err
	}
	return uninstalledRelease.Release, nil
}

type ApplyOptions struct {
	Version string
	Repo    string
	DryRun  bool
}

// chartName is the 'release name' whenever.
func (h *Helm) ApplyChart(ctx context.Context,
	releaseName, releaseNamespace string, chartNameOrPath string,
	values map[string]interface{}, options ApplyOptions,
) (*release.Release, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("name", releaseName, "namespace", releaseNamespace, "repo", options.Repo, "version", options.Version)

	if chartNameOrPath == "" {
		chartNameOrPath = releaseName
	}

	version, repo := options.Version, options.Repo
	log.Info("loading chart")
	chart, err := LoadChart(ctx, chartNameOrPath, repo, version)
	if err != nil {
		return nil, err
	}
	if releaseName == "" {
		releaseName = chart.Name()
	}

	// do dry run
	if options.DryRun {
		log.Info("dry run installing")
		install := action.NewInstall(&action.Configuration{})
		install.ReleaseName = releaseName
		install.Namespace = releaseNamespace
		install.DryRun, install.DisableHooks, install.ClientOnly, install.CreateNamespace = true, true, true, true
		return install.Run(chart, values)
	}

	cfg, err := NewHelmConfig(releaseNamespace, h.Config)
	if err != nil {
		return nil, err
	}
	existRelease, err := action.NewGet(cfg).Run(releaseName)
	if err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, err
		}
		log.Info("installing")
		install := action.NewInstall(cfg)
		install.ReleaseName = releaseName
		install.CreateNamespace = true
		install.Namespace = releaseNamespace
		install.ClientOnly = options.DryRun
		return install.Run(chart, values)
	}
	// check should upgrade
	if existRelease.Info.Status == release.StatusDeployed && reflect.DeepEqual(existRelease.Config, values) {
		log.Info("already deployed and no values changed")
		return existRelease, nil
	}
	log.Info("upgrading")
	client := action.NewUpgrade(cfg)
	client.Namespace = releaseNamespace
	client.ResetValues = true
	client.DryRun = options.DryRun
	return client.Run(releaseName, chart, values)
}

func NewHelmConfig(namespace string, restConfig *rest.Config) (*action.Configuration, error) {
	log := func(format string, v ...interface{}) {
	}
	cligetter, err := NewRESTClientGetter(restConfig)
	if err != nil {
		return nil, err
	}
	config := &action.Configuration{}
	config.Init(cligetter, namespace, "", log) // release storage namespace
	if kc, ok := config.KubeClient.(*kube.Client); ok {
		kc.Namespace = namespace // install to namespace
	}
	return config, nil
}

// name is the name of the chart
// repo is the url of the chart repository,eg: http://charts.example.com
// if repopath is not empty,download it from repo and set chartNameOrPath to repo/repopath.
// LoadChart loads the chart from the repository
func LoadChart(ctx context.Context, name, repo, version string) (*chart.Chart, error) {
	_, chart, err := DownloadChart(ctx, repo, name, version)
	if err != nil {
		return nil, fmt.Errorf("download chart: %w", err)
	}
	return chart, nil
}

func DownloadChart(ctx context.Context, repo, name, version string) (string, *chart.Chart, error) {
	chartPathOptions := action.ChartPathOptions{RepoURL: repo, Version: version}
	settings := cli.New()
	chartPath, err := chartPathOptions.LocateChart(name, settings)
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
			Out:              io.Discard,
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
	return chartPath, chart, err
}
