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

type HelmApplyer struct {
	ChartsDir string `json:"chartsDir,omitempty"`
}

// nolint: funlen,nestif
func (r *HelmApplyer) Apply(ctx context.Context, plugin pluginsv1beta1.InstallerSpecPlugin, status *pluginsv1beta1.InstallerStatusStatus) error {
	namespace, name := plugin.Namespace, plugin.Name
	log := logr.FromContext(ctx).WithValues("name", name, "namespace", namespace)

	config := NewHelmConfig(namespace)

	// do get first
	isNotFound := false
	exist, err := action.NewGet(config).Run(name)
	if err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			return err
		}
		isNotFound = true
	}

	if !plugin.Enabled {
		if isNotFound {
			if status.Status == pluginsv1beta1.StatusUninstalled || status.Status == pluginsv1beta1.StatusNotInstall {
				return nil
			}
			log.Info("plugin is disabled and not installed")
			status.Status = pluginsv1beta1.StatusNotInstall
			return nil
		}
		// record exist and uninstalled
		if exist.Info.Status == release.StatusUninstalled {
			log.Info("plugin is disabled and uninstalled")
			updateStatusFromRelease(status, exist)
			return nil
		}
		// uninstall
		log.Info("uninstalling")
		uninstall := action.NewUninstall(config)
		uninstalledRelease, err := uninstall.Run(name)
		if err != nil {
			log.Error(err, "on uninstall")
			return err
		}
		log.Info("uninstalled")
		if urelease := uninstalledRelease.Release; urelease != nil {
			updateStatusFromRelease(status, urelease)
		} else {
			log.Info("uninstalled, but no release")
			status.Message = uninstalledRelease.Info
			status.Status = pluginsv1beta1.StatusUninstalled
		}
		return nil
	}

	values := UnmarshalValues(plugin.Values) // ignore error
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

	if isNotFound {
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
		updateStatusFromRelease(status, installedRelease)
		return nil
	}

	// should upgrade
	if exist.Info.Status == release.StatusDeployed && reflect.DeepEqual(exist.Config, values) {
		log.V(5).Info("already uptodate")
		updateStatusFromRelease(status, exist)
		return nil
	}

	// upgrade
	log.Info("upgrading")
	log.V(5).Info("status not healthy", "status", exist.Info.Status)
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
	updateStatusFromRelease(status, upgradeRelease)
	return nil
}

func updateStatusFromRelease(status *pluginsv1beta1.InstallerStatusStatus, release *release.Release) {
	if release == nil {
		return
	}
	resetStatus(status)

	status.Status = pluginsv1beta1.Status(release.Info.Status)
	status.Message = release.Info.Description

	// https://github.com/golang/go/issues/19502
	// metav1.Time and time.Time are not comparable directly
	convtime := func(t time.Time) metav1.Time {
		t, _ = time.Parse(time.RFC3339, t.Format(time.RFC3339))
		return metav1.Time{Time: t}
	}

	status.CreationTimestamp = convtime(release.Info.FirstDeployed.Time)
	status.UpgradeTimestamp = convtime(release.Info.LastDeployed.Time)
	status.Notes = release.Info.Notes
	status.Version = release.Chart.Metadata.Version
	status.Values = MarshalValues(release.Config)

	if !release.Info.Deleted.IsZero() {
		status.DeletionTimestamp = func() *metav1.Time {
			time := convtime(release.Info.Deleted.Time)
			return &time
		}()
	}
}

func resetStatus(status *pluginsv1beta1.InstallerStatusStatus) {
	*status = pluginsv1beta1.InstallerStatusStatus{Name: status.Name, Namespace: status.Namespace}
}

func extractCurrentStatus(plugin pluginsv1beta1.InstallerSpecPlugin, installer *pluginsv1beta1.Installer) *pluginsv1beta1.InstallerStatusStatus {
	for i, exist := range installer.Status.States {
		if exist.Name != plugin.Name || exist.Namespace != plugin.Namespace {
			continue
		}
		return &installer.Status.States[i]
	}
	installer.Status.States = append(installer.Status.States, pluginsv1beta1.InstallerStatusStatus{
		Name:      plugin.Name,
		Namespace: plugin.Namespace,
	})
	status := &installer.Status.States[len(installer.Status.States)-1]
	return status
}

func updatedStatus(
	plugin pluginsv1beta1.InstallerSpecPlugin,
	installer *pluginsv1beta1.Installer,
	status *pluginsv1beta1.InstallerStatusStatus,
) bool {
	name, namespace := plugin.Name, plugin.Namespace
	if status != nil {
		status.Name = name
		status.Namespace = namespace
	}
	for i, exist := range installer.Status.States {
		if exist.Name != name || exist.Namespace != namespace {
			continue
		}
		if status == nil {
			installer.Status.States = append(installer.Status.States[:i], installer.Status.States[i+1:]...)
			return true
		}
		if reflect.DeepEqual(exist, *status) {
			return false
		}
		installer.Status.States[i] = *status
		return true
	}
	if status != nil {
		installer.Status.States = append(installer.Status.States, *status)
		return true
	}
	return false
}
