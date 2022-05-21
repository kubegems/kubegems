package clusterhandler

import (
	"context"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"kubegems.io/pkg/installer/controllers/helm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	KubeGemPluginsPath               = "plugins"
	KubeGemsInstallerPluginName      = "kubegems-installer"
	KubeGemsInstallerPluginNamespace = "kubegems-installer"
	KubeGemsLocalPluginsNamespace    = "kubegems-local"
	KubeGemsLocalPluginsName         = "kubegems-local-stack"
)

type OpratorInstaller struct {
	Config           *rest.Config           // target cluster config
	Version          string                 // version of the installer chart
	InstallNamespace string                 // namespace where the local-components is installed
	PluginsValues    map[string]interface{} // values pass to `plugins-local-template.yaml`
}

func (i OpratorInstaller) Apply(ctx context.Context) error {
	chartpath := filepath.Join(KubeGemPluginsPath, KubeGemsInstallerPluginName)
	log.FromContextOrDiscard(ctx).Info("applying kubegems-installer chart", "chartPath", chartpath)

	relese, err := (&helm.Helm{Config: i.Config}).ApplyChart(ctx,
		KubeGemsInstallerPluginName, KubeGemsInstallerPluginNamespace,
		chartpath, i.PluginsValues, helm.ApplyOptions{},
	)
	if err != nil {
		return err
	}
	log.Info("apply kubegems installer succeed", "namespace", relese.Namespace, "name", relese.Name, "version", relese.Version)

	if i.InstallNamespace == "" {
		i.InstallNamespace = KubeGemsLocalPluginsNamespace
	}

	allinoneplugin := &pluginsv1beta1.Plugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: pluginsv1beta1.GroupVersion.String(),
			Kind:       "Plugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeGemsLocalPluginsName,
			Namespace: i.InstallNamespace,
		},
		Spec: pluginsv1beta1.PluginSpec{
			Enabled:          true,
			Kind:             pluginsv1beta1.PluginKindTemplate,
			InstallNamespace: i.InstallNamespace, // set ns to install ns
			Values:           pluginsv1beta1.Values{Object: i.PluginsValues},
		},
	}
	if err := kube.Apply(ctx, i.Config, []client.Object{allinoneplugin}, kube.WithCreateNamespace()); err != nil {
		return err
	}
	return nil
}

func (i OpratorInstaller) Remove(ctx context.Context) error {
	if i.InstallNamespace == "" {
		i.InstallNamespace = KubeGemsLocalPluginsNamespace
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
