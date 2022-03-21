package controllers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
)

type HelmApplier struct {
	ChartsDir string `json:"chartsDir,omitempty"`
}

func (r *HelmApplier) Apply(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	namespace, name := plugin.Namespace, plugin.Name
	if plugin.Repo == "" {
		// if no remote repo found,use local charts
		plugin.Repo = "file://" + r.ChartsDir
	}

	upgradeRelease, err := ApplyChart(ctx, name, namespace, plugin.Version, plugin.Values, plugin.Repo)
	if err != nil {
		return err
	}

	if upgradeRelease.Info.Status != release.StatusDeployed {
		status.Notes = upgradeRelease.Info.Notes
		return fmt.Errorf("apply not finished:%s", upgradeRelease.Info.Description)
	}

	status.Name, status.Namespace = upgradeRelease.Name, upgradeRelease.Namespace
	status.Phase = pluginsv1beta1.PluginPhaseInstalled
	status.Message = upgradeRelease.Info.Description
	status.Version = upgradeRelease.Chart.Metadata.Version
	status.CreationTimestamp = convtime(upgradeRelease.Info.FirstDeployed.Time)
	status.UpgradeTimestamp = convtime(upgradeRelease.Info.LastDeployed.Time)
	status.Notes = upgradeRelease.Info.Notes
	status.Values = upgradeRelease.Config
	return nil
}

func (r *HelmApplier) Remove(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	log := logr.FromContextOrDiscard(ctx)
	namespace, name := plugin.Namespace, plugin.Name

	if status.Phase == pluginsv1beta1.PluginPhaseNone {
		log.Info("already removed")
		return nil
	}
	if status.Phase == "" {
		status.Phase = pluginsv1beta1.PluginPhaseNone
		status.Message = "plugin not install"
		return nil
	}

	// uninstall
	release, err := RemoveChart(ctx, name, namespace)
	if err != nil {
		return err
	}
	if release == nil {
		status.Phase = pluginsv1beta1.PluginPhaseNone
		status.Message = "plugin not install"
		return nil
	}

	status.Phase = pluginsv1beta1.PluginPhaseNone
	status.Message = release.Info.Description
	status.DeletionTimestamp = convtime(release.Info.Deleted.Time)
	status.Notes = release.Info.Notes
	status.Values = release.Config
	return nil
}

// https://github.com/golang/go/issues/19502
// metav1.Time and time.Time are not comparable directly
func convtime(t time.Time) metav1.Time {
	t, _ = time.Parse(time.RFC3339, t.Format(time.RFC3339))
	return metav1.Time{Time: t}
}

func RemoveChart(ctx context.Context, chartName, installNamespace string) (*release.Release, error) {
	log := logr.FromContext(ctx).WithValues("name", chartName, "namespace", installNamespace)

	releaseName := chartName
	cfg := NewHelmConfig(installNamespace)

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

func ApplyChart(ctx context.Context, chartName, installNamespace string, version string, values map[string]interface{}, repo string) (*release.Release, error) {
	log := logr.FromContextOrDiscard(ctx)

	releaseName := chartName
	cfg := NewHelmConfig(installNamespace)

	existRelease, err := action.NewGet(cfg).Run(releaseName)
	if err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, err
		}

		chart, err := LoadChart(chartName, version, repo)
		if err != nil {
			return nil, err
		}

		log.Info("installing")
		install := action.NewInstall(cfg)
		install.ReleaseName = releaseName
		install.CreateNamespace = true
		install.Namespace = installNamespace
		return install.Run(chart, values)
	}

	// check should upgrade
	if existRelease.Info.Status == release.StatusDeployed && reflect.DeepEqual(existRelease.Config, values) {
		log.Info("already deployed and no values changed")
		return existRelease, nil
	}

	chart, err := LoadChart(chartName, version, repo)
	if err != nil {
		return nil, err
	}

	log.Info("upgrading")
	client := action.NewUpgrade(cfg)
	client.Namespace = installNamespace
	return client.Run(releaseName, chart, values)
}

func NewHelmConfig(namespace string) *action.Configuration {
	getter := genericclioptions.NewConfigFlags(true)
	getter.Namespace = pointer.String(namespace) // must set to ns to install chart
	config := &action.Configuration{}
	config.Init(getter, namespace, "", func(format string, v ...interface{}) {
	})
	return config
}

// name is the name of the chart
// repo is the url of the chart repository,eg: http://charts.example.com or file:///app/charts
// version is the version of the chart,ignored when repo is file://
// LoadChart loads the chart from the repo
func LoadChart(name string, version string, repo string) (*chart.Chart, error) {
	settings := cli.New()
	chartPathOptions := action.ChartPathOptions{
		RepoURL: repo,
		Version: version,
	}

	repoURL, err := url.Parse(repo)
	if err != nil {
		return nil, err
	}

	if repoURL.Scheme == "file" {
		name = filepath.Join(repoURL.Path, name)
	} else {
		chartPathOptions.RepoURL = repo
	}

	getters := getter.All(settings)

	chartPath, err := chartPathOptions.LocateChart(name, settings)
	if err != nil {
		return nil, err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}

	// nolint: nestif
	if deps := chart.Metadata.Dependencies; deps != nil {
		if err := action.CheckDependencies(chart, deps); err != nil {
			// dependencies update
			if true {
				man := &downloader.Manager{
					Out:              io.Discard,
					ChartPath:        chartPath,
					Keyring:          chartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          getters,
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
					Debug:            settings.Debug,
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
				// Reload the chart with the updated Chart.lock file.
				if chart, err = loader.Load(chartPath); err != nil {
					return nil, fmt.Errorf("failed reloading chart after repo update:%w", err)
				}
			} else {
				return nil, err
			}
		}
	}
	return chart, nil
}
