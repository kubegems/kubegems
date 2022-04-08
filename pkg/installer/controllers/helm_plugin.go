package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	kubeutils "github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	if err := DownloadPlugin(ctx, &plugin, r.ChartsDir); err != nil {
		return err
	}

	upgradeRelease, err := r.Helm.ApplyChart(ctx, plugin.Name, plugin.Namespace, plugin.Path, plugin.Values, ApplyOptions{
		Version: plugin.Version, Repo: plugin.Repo, DryRun: plugin.DryRun,
	})
	if err != nil {
		return err
	}

	ress, _ := parseResources(upgradeRelease.Manifest)
	status.Resources = ress

	if !plugin.DryRun && (upgradeRelease.Info.Status != release.StatusDeployed) {
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
	release, err := r.Helm.RemoveChart(ctx, plugin.Name, plugin.Namespace, RemoveOptions{
		DryRun: plugin.DryRun,
	})
	if err != nil {
		return err
	}

	ress, _ := parseResources(release.Manifest)
	status.Resources = ress

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

func parseResources(manifests string) ([]*unstructured.Unstructured, error) {
	return kubeutils.SplitYAML([]byte(manifests))
}
