package clusterhandler

import (
	"context"
	"path/filepath"

	"k8s.io/client-go/rest"
	"kubegems.io/pkg/installer/controllers"
	"kubegems.io/pkg/log"
)

const (
	KubeGemInstallerChartName      = "kubegems-installer"
	KubeGemInstallerChartPath      = "/app/charts"
	KubeGemInstallerChartNamespace = "kubegems-installer"
)

type OpratorInstaller struct {
	ChartPath string                 // path to the chart
	Config    *rest.Config           // target cluster config
	Version   string                 // version of the chart
	Values    map[string]interface{} // values pass to kubegems-installer chart value.yaml
}

func (i OpratorInstaller) Apply(ctx context.Context) error {
	if i.ChartPath == "" {
		i.ChartPath = KubeGemInstallerChartPath
	}
	log.FromContextOrDiscard(ctx).Info("applying kubegems-installer chart",
		"values", i.Values,
		"version", i.Version,
		"chartPath", i.ChartPath,
	)
	helm := controllers.Helm{Config: i.Config}
	path, err := filepath.Abs(i.ChartPath)
	if err != nil {
		return err
	}
	relese, err := helm.ApplyChart(ctx, KubeGemInstallerChartName, KubeGemInstallerChartNamespace,
		controllers.ApplyOptions{Values: i.Values, Repo: "file://" + path},
	)
	if err != nil {
		return err
	}
	_ = relese
	return nil
}

func (i OpratorInstaller) Remove(ctx context.Context) error {
	log.FromContextOrDiscard(ctx).Info("removing kubegems-installer chart")
	helm := controllers.Helm{Config: i.Config}
	relese, err := helm.RemoveChart(ctx, KubeGemInstallerChartName, KubeGemInstallerChartNamespace)
	if err != nil {
		return err
	}
	_ = relese
	return nil
}
