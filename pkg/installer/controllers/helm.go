package controllers

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
)

func NewHelmConfig(namespace string) *action.Configuration {
	getter := genericclioptions.NewConfigFlags(true)
	getter.Namespace = pointer.String(namespace) // must set to ns to install chart
	config := &action.Configuration{}
	config.Init(getter, namespace, "", func(format string, v ...interface{}) {
	})
	return config
}

type HelmApplier struct {
	ChartsDir string `json:"chartsDir,omitempty"`
}

// nolint: funlen
func (r *HelmApplier) Apply(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	namespace, name := plugin.Namespace, plugin.Name
	log := logr.FromContext(ctx).WithValues("name", name, "namespace", namespace)

	config := NewHelmConfig(namespace)
	values := plugin.Values

	loader, err := loader.Loader(filepath.Join(r.ChartsDir, name))
	if err != nil {
		log.Error(err, "on chart loader")
		return err
	}
	chart, err := loader.Load()
	if err != nil {
		log.Error(err, "on chart load")
		return err
	}

	exist, err := action.NewGet(config).Run(name)
	if err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			return err
		}
		// install
		log.Info("installing")
		install := action.NewInstall(config)
		install.ReleaseName = name
		install.CreateNamespace = true
		install.Namespace = namespace
		installedRelease, err := install.Run(chart, values)
		if err != nil {
			log.Info("on install")
			return err
		}
		log.Info("installed")

		status.Name, status.Namespace = installedRelease.Name, installedRelease.Namespace
		status.Phase = pluginsv1beta1.StatusDeployed
		status.CreationTimestamp = convtime(installedRelease.Info.FirstDeployed.Time)
		status.UpgradeTimestamp = convtime(installedRelease.Info.LastDeployed.Time)
		status.Notes = installedRelease.Info.Notes
		status.Values = installedRelease.Config
		return nil
	}

	// should upgrade
	if exist.Info.Status == release.StatusDeployed && reflect.DeepEqual(exist.Config, values) {
		log.V(5).Info("already uptodate")
		return nil
	}

	// upgrade
	log.Info("upgrading")
	log.V(5).Info("status", "status", exist.Info.Status)
	log.V(5).Info("values diff", "old", exist.Config, "new", values)
	upgrade := action.NewUpgrade(config)
	upgrade.Install = true
	upgrade.Namespace = namespace
	upgradeRelease, err := upgrade.Run(name, chart, values)
	if err != nil {
		log.Error(err, "on upgrade")
		return err
	}
	log.Info("upgraded")

	now := metav1.Now()
	status.Name, status.Namespace = upgradeRelease.Name, upgradeRelease.Namespace
	status.Phase = pluginsv1beta1.StatusDeployed
	status.Message = upgradeRelease.Info.Description
	status.UpgradeTimestamp = now
	status.Notes = upgradeRelease.Info.Notes
	status.Values = upgradeRelease.Config
	return nil
}

func (r *HelmApplier) Remove(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	namespace, name := plugin.Namespace, plugin.Name
	log := logr.FromContext(ctx).WithValues("name", name, "namespace", namespace)

	config := NewHelmConfig(namespace)

	exist, err := action.NewGet(config).Run(name)
	if err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			return err
		}
		if status.Phase == pluginsv1beta1.StatusUninstalled ||
			status.Phase == pluginsv1beta1.StatusNotInstall {
			return nil
		}
		log.Info("plugin is not installed")
		status.Phase = pluginsv1beta1.StatusNotInstall
		status.Message = "plugin is not installed"
		return nil
	}

	// check if the plugin is installed by current plugin
	if status.Phase != pluginsv1beta1.StatusDeployed && status.Phase != pluginsv1beta1.StatusFailed {
		log.Info("plugin is not deployed but this plugin but requested to uninstall")
		status.Phase = pluginsv1beta1.StatusUnknown
		status.Message = "plugin is not deployed by current plugin"
		return nil
	}

	// uninstall
	log.Info("uninstalling")
	uninstall := action.NewUninstall(config)
	uninstalledRelease, err := uninstall.Run(exist.Name)
	if err != nil {
		log.Error(err, "on uninstall")
		return err
	}
	log.Info("uninstalled")
	status.Phase = pluginsv1beta1.StatusUninstalled
	status.Message = uninstalledRelease.Info
	if urelease := uninstalledRelease.Release; urelease != nil {
		status.DeletionTimestamp = convtime(urelease.Info.Deleted.Time)
		status.Notes = urelease.Info.Notes
		status.Values = urelease.Config
	}
	return nil
}

// https://github.com/golang/go/issues/19502
// metav1.Time and time.Time are not comparable directly
func convtime(t time.Time) metav1.Time {
	t, _ = time.Parse(time.RFC3339, t.Format(time.RFC3339))
	return metav1.Time{Time: t}
}
