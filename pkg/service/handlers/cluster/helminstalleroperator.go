package clusterhandler

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"kubegems.io/pkg/installer/controllers"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	KubeGemInstallerChartPath      = "/app/charts"
	KubeGemInstallerChartName      = "kubegems-installer"
	KubeGemInstallerChartNamespace = "kubegems-installer"
	KubeGemLocalPluginsNamespace   = "kubegems-local"
	KubeGemLocalPluginsNmae        = "kubegems-local-stack"
)

type OpratorInstaller struct {
	Config           *rest.Config           // target cluster config
	Version          string                 // version of the installer chart
	InstallNamespace string                 // namespace where the local-components is installed
	PluginsValues    map[string]interface{} // values pass to `plugins-local-template.yaml`
}

func (i OpratorInstaller) Apply(ctx context.Context) error {
	if i.InstallNamespace == "" {
		i.InstallNamespace = KubeGemLocalPluginsNamespace
	}
	chartpath := KubeGemInstallerChartPath
	log.FromContextOrDiscard(ctx).Info("applying kubegems-installer chart", "chartPath", chartpath)
	relese, err := (&controllers.Helm{Config: i.Config}).ApplyChart(ctx, KubeGemInstallerChartName, KubeGemInstallerChartNamespace, "file://"+chartpath,
		controllers.ApplyOptions{
			Values: i.PluginsValues,
			Path:   KubeGemInstallerChartName,
		},
	)
	if err != nil {
		return err
	}
	_ = relese

	allinoneplugin := &pluginsv1beta1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeGemLocalPluginsNmae,
			Namespace: i.InstallNamespace,
		},
		Spec: pluginsv1beta1.PluginSpec{
			Enabled:          true,
			InstallNamespace: i.InstallNamespace, // set ns to install ns
			Values:           controllers.MarshalValues(i.PluginsValues),
		},
	}
	if err := kube.Apply(ctx, i.Config, []client.Object{allinoneplugin}, kube.WithCreateNamespace()); err != nil {
		return err
	}
	return nil
}

func (i OpratorInstaller) Remove(ctx context.Context) error {
	if i.InstallNamespace == "" {
		i.InstallNamespace = KubeGemInstallerChartNamespace
	}

	// remove all plugins
	log.FromContextOrDiscard(ctx).Info("removing kubegems plugins")
	plugins := &pluginsv1beta1.PluginList{}
	cli, err := kube.NewClient(i.Config)
	if err != nil {
		return err
	}
	if err := cli.List(ctx, plugins, client.InNamespace(i.InstallNamespace)); err != nil {
		return err
	}
	for _, plugin := range plugins.Items {
		if err := cli.Delete(ctx, &plugin); err != nil {
			return err
		}
	}

	// wait all plugins removed,then remove charts

	// need to remove installer-operator by hand

	// log.FromContextOrDiscard(ctx).Info("removing kubegems-installer chart")
	// helm := controllers.Helm{Config: i.Config}
	// relese, err := helm.RemoveChart(ctx, KubeGemInstallerChartName, i.InstallNamespace)
	// if err != nil {
	// 	return err
	// }
	// _ = relese
	return nil
}
