package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/helm"
)

type HelmApplier struct {
	CacheDir string
	Helm     *helm.Helm
}

func NewHelm(config *rest.Config, cache string) *HelmApplier {
	return &HelmApplier{
		Helm:     &helm.Helm{Config: config},
		CacheDir: cache,
	}
}

func (r *HelmApplier) Template(ctx context.Context, bundle *pluginsv1beta1.Plugin, into string) ([]byte, error) {
	rls := r.getPreRelease(bundle)
	upgradeRelease, err := r.Helm.ApplyChart(
		ctx, rls.Name, rls.Namespace, into, rls.Config,
		helm.ApplyOptions{DryRun: true})
	if err != nil {
		return nil, err
	}
	return []byte(upgradeRelease.Manifest), nil
}

func (r *HelmApplier) Apply(ctx context.Context, bundle *pluginsv1beta1.Plugin, into string) error {
	rls := r.getPreRelease(bundle)
	applyedRelease, err := r.Helm.ApplyChart(ctx, rls.Name, rls.Namespace, into, rls.Config, helm.ApplyOptions{})
	if err != nil {
		return err
	}
	if applyedRelease.Info.Status != release.StatusDeployed {
		return fmt.Errorf("apply not finished:%s", applyedRelease.Info.Description)
	}
	bundle.Status.Phase = pluginsv1beta1.PluginPhaseInstalled
	bundle.Status.Message = applyedRelease.Info.Description
	bundle.Status.InstallNamespace = applyedRelease.Namespace
	bundle.Status.CreationTimestamp = convtime(applyedRelease.Info.FirstDeployed.Time)
	bundle.Status.UpgradeTimestamp = convtime(applyedRelease.Info.LastDeployed.Time)
	bundle.Status.Values = pluginsv1beta1.Values{Object: applyedRelease.Config}
	bundle.Status.Version = applyedRelease.Chart.Metadata.Version
	bundle.Status.Notes = applyedRelease.Info.Notes
	return nil
}

func (r *HelmApplier) Remove(ctx context.Context, bundle *pluginsv1beta1.Plugin) error {
	log := logr.FromContextOrDiscard(ctx)
	if bundle.Status.Phase == pluginsv1beta1.PluginPhaseNone || bundle.Status.Phase == "" {
		log.Info("already removed or not installed")
		return nil
	}
	rls := r.getPreRelease(bundle)
	// uninstall
	removedRelease, err := r.Helm.RemoveChart(ctx, rls.Name, rls.Namespace, helm.RemoveOptions{})
	if err != nil {
		return err
	}
	log.Info("removed")
	if removedRelease == nil {
		bundle.Status.Phase = pluginsv1beta1.PluginPhaseNone
		bundle.Status.Message = "plugin not install"
		return nil
	}
	bundle.Status.Phase = pluginsv1beta1.PluginPhaseNone
	bundle.Status.Message = removedRelease.Info.Description
	bundle.Status.Notes = removedRelease.Info.Notes
	bundle.Status.DeletionTimestamp = func() *metav1.Time {
		t := convtime(removedRelease.Info.Deleted.Time)
		return &t
	}()
	return nil
}

func (r HelmApplier) getPreRelease(bundle *pluginsv1beta1.Plugin) *release.Release {
	releaseNamespace := bundle.Spec.InstallNamespace
	if releaseNamespace == "" {
		releaseNamespace = bundle.Namespace
	}
	releaseName := bundle.Name
	values := bundle.Spec.Values.Object
	return &release.Release{Name: releaseName, Namespace: releaseNamespace, Config: values}
}

// https://github.com/golang/go/issues/19502
// metav1.Time and time.Time are not comparable directly
func convtime(t time.Time) metav1.Time {
	t, _ = time.Parse(time.RFC3339, t.Format(time.RFC3339))
	return metav1.Time{Time: t}
}
