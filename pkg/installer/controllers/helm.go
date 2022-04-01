package controllers

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
)

type HelmPlugin struct {
	Helm      *Helm  `json:"helm,omitempty"`
	ChartsDir string `json:"chartsDir,omitempty"`
}

func NewHelmPlugin(config *rest.Config, path string) *HelmPlugin {
	if abs, _ := filepath.Abs(path); abs != path {
		path = abs
	}
	return &HelmPlugin{Helm: &Helm{Config: config}, ChartsDir: path}
}

func (r *HelmPlugin) Apply(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	repo, path := plugin.Repo, plugin.Path
	if repo == "" {
		repo = "file://" + r.ChartsDir
		if path == "" {
			path = plugin.Name
		}
	}

	upgradeRelease, err := r.Helm.ApplyChart(ctx, plugin.Name, plugin.Namespace, repo, ApplyOptions{
		Version: plugin.Version,
		Path:    path,
		Values:  plugin.Values,
	})
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

func (r *HelmPlugin) Remove(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	log := logr.FromContextOrDiscard(ctx)

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
	release, err := r.Helm.RemoveChart(ctx, plugin.Name, plugin.Namespace)
	if err != nil {
		return err
	}
	if release == nil {
		status.Phase = pluginsv1beta1.PluginPhaseNone
		status.Message = "plugin not install"
		return nil
	}

	status.Phase = pluginsv1beta1.PluginPhaseRemoved
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

type Helm struct {
	Config *rest.Config
}

func (h *Helm) RemoveChart(ctx context.Context, chartName, installNamespace string) (*release.Release, error) {
	log := logr.FromContextOrDiscard(ctx)

	releaseName := chartName
	cfg, err := NewHelmConfig(installNamespace, h.Config)
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
	Path    string
	Values  map[string]interface{}
}

// ApplyChart applies a chart to a release.
// To install a local chart,set the path to the chart and chart name is ignored when find chart
// 			eg: name: local-path-provisioner
// 				repo: ""
// 				path: tmp/charts/local-path-provisioner  // local path to the chart.
//			or:
// 				name: local-path-provisioner
// 				repo: "file:///tmp/charts"   // must absolute path
// 				path: "local-path-provisioner"
//
// To install an in git chart,set repo to git clone url set version to git branch/tag and set path to chart directory in repo.
// if path is git root,set path to '.'
// 			eg: name: local-path-provisioner
// 				repo: https://github.com/rancher/local-path-provisioner.git
// 				version: master
// 				path: deploy/chart/local-path-provisioner
//
// To install a normal remote chart,set repo to chart repository url and set version to chart version.(keep 'path' empty)
// 			eg: name: mysql
//				repo: https://charts.bitnami.com/bitnami
// 				version: 1.0.0
// 				name: mysql
//
// chartName is the 'release name' whenever.
func (h *Helm) ApplyChart(ctx context.Context, name, namespace string, repo string, options ApplyOptions) (*release.Release, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("name", name, "namespace", namespace, "repo", repo, "version", options.Version, "path", options.Path)

	version, path, values := options.Version, options.Path, options.Values
	releaseName := name

	cfg, err := NewHelmConfig(namespace, h.Config)
	if err != nil {
		return nil, err
	}

	existRelease, err := action.NewGet(cfg).Run(releaseName)
	if err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, err
		}

		log.Info("loading chart")
		chart, err := LoadChart(ctx, name, version, repo, path)
		if err != nil {
			return nil, err
		}

		log.Info("installing")
		install := action.NewInstall(cfg)
		install.ReleaseName = releaseName
		install.CreateNamespace = true
		install.Namespace = namespace
		return install.Run(chart, values)
	}

	// check should upgrade
	if existRelease.Info.Status == release.StatusDeployed && reflect.DeepEqual(existRelease.Config, values) {
		log.Info("already deployed and no values changed")
		return existRelease, nil
	}

	chart, err := LoadChart(ctx, name, version, repo, path)
	if err != nil {
		return nil, err
	}

	log.Info("upgrading")
	client := action.NewUpgrade(cfg)
	client.Namespace = namespace
	client.ResetValues = true
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
func LoadChart(ctx context.Context, chartNameOrPath string, version string, repo, path string) (*chart.Chart, error) {
	if path != "" {
		p, err := Download(ctx, repo, version, path)
		if err != nil {
			return nil, fmt.Errorf("download chart: %w", err)
		}
		chartNameOrPath = p
	}

	chartPathOptions := action.ChartPathOptions{RepoURL: repo, Version: version}
	settings := cli.New()
	chartPath, err := chartPathOptions.LocateChart(chartNameOrPath, settings)
	if err != nil {
		return nil, err
	}
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}
	if err := action.CheckDependencies(chart, chart.Metadata.Dependencies); err != nil {
		// dependencies update
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
			return nil, err
		}
		// Reload the chart with the updated Chart.lock file.
		if chart, err = loader.Load(chartPath); err != nil {
			return nil, fmt.Errorf("failed reloading chart after repo update:%w", err)
		}
	}
	return chart, nil
}
