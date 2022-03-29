package clusterhandler

import (
	"context"
	"path/filepath"

	"k8s.io/client-go/rest"
	"kubegems.io/pkg/installer/controllers"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/kube"
)

const (
	KubeGemInstallerChartPath      = "/app/charts"
	KubeGemInstallerChartName      = "kubegems-installer"
	KubeGemInstallerChartNamespace = "kubegems-installer"

	KubeGemInstallerPluginsPath  = "/app/plugins"
	KubeGemLocalPluginsName      = "kubegems-local-plugins"
	KubeGemLocalPluginsNamespace = "kubegems-local"
)

type OpratorInstaller struct {
	ChartPath        string                 // path to the chart
	Config           *rest.Config           // target cluster config
	Version          string                 // version of the installer chart
	InstallNamespace string                 // namespace where the installer is installed
	PluginsValues    map[string]interface{} // values pass to `plugins-local-template.yaml`
}

func (i OpratorInstaller) Apply(ctx context.Context) error {
	if i.ChartPath == "" {
		i.ChartPath = KubeGemInstallerChartPath
	}
	if i.InstallNamespace == "" {
		i.InstallNamespace = KubeGemInstallerChartNamespace
	}
	log.FromContextOrDiscard(ctx).Info("applying kubegems-installer chart", "chartPath", i.ChartPath)
	helm := controllers.Helm{Config: i.Config}
	path, err := filepath.Abs(i.ChartPath)
	if err != nil {
		return err
	}
	relese, err := helm.ApplyChart(ctx, KubeGemInstallerChartName, i.InstallNamespace, "",
		controllers.ApplyOptions{
			Values: i.PluginsValues,
			Path:   path,
		},
	)
	if err != nil {
		return err
	}
	_ = relese
	// apply plugins
	templatepath := filepath.Join(KubeGemInstallerPluginsPath, KubeGemLocalPluginsName)
	resources, err := controllers.ParseManifests(templatepath, i.PluginsValues)
	if err != nil {
		return err
	}
	if err := kube.Apply(ctx, i.Config, resources, kube.WithCreateNamespace()); err != nil {
		return err
	}
	return nil
}

func (i OpratorInstaller) Remove(ctx context.Context) error {
	if i.InstallNamespace == "" {
		i.InstallNamespace = KubeGemInstallerChartNamespace
	}
	log.FromContextOrDiscard(ctx).Info("removing kubegems-installer chart")
	helm := controllers.Helm{Config: i.Config}
	relese, err := helm.RemoveChart(ctx, KubeGemInstallerChartName, i.InstallNamespace)
	if err != nil {
		return err
	}
	_ = relese
	return nil
}
