package clusterhandler

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"kubegems.io/pkg/installer/controllers"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	KubeGemInstallerChartPath      = "/app/charts"
	KubeGemInstallerChartName      = "kubegems-installer"
	KubeGemInstallerChartNamespace = "kubegems-installer"
	KubeGemLocalPluginsNamespace   = "kubegems-local"
	KubeGemLocalPluginsFile        = "/app/plugins/kubegems-local-plugins.yaml"
)

type OpratorInstaller struct {
	ChartsPath           string                 // path to the charts
	LocalstackPluginFile string                 // path to the plugins kubegems-local-stack.yaml
	Config               *rest.Config           // target cluster config
	Version              string                 // version of the installer chart
	InstallNamespace     string                 // namespace where the local-components is installed
	PluginsValues        map[string]interface{} // values pass to `plugins-local-template.yaml`
}

func (i OpratorInstaller) Apply(ctx context.Context) error {
	if i.ChartsPath == "" {
		i.ChartsPath = KubeGemInstallerChartPath
	}
	if i.InstallNamespace == "" {
		i.InstallNamespace = KubeGemLocalPluginsNamespace
	}
	log.FromContextOrDiscard(ctx).Info("applying kubegems-installer chart", "chartPath", i.ChartsPath)
	helm := controllers.Helm{Config: i.Config}

	repopath, err := filepath.Abs(i.ChartsPath)
	if err != nil {
		return err
	}
	relese, err := helm.ApplyChart(ctx, KubeGemInstallerChartName, KubeGemInstallerChartNamespace, "file://"+repopath,
		controllers.ApplyOptions{
			Values: i.PluginsValues,
			Path:   KubeGemInstallerChartName,
		},
	)
	if err != nil {
		return err
	}
	_ = relese

	if i.LocalstackPluginFile == "" {
		i.LocalstackPluginFile = KubeGemLocalPluginsFile
	}
	// apply allinone plugin: deploy/plugins/plugins-local-stack.yaml
	content, err := ioutil.ReadFile(i.LocalstackPluginFile)
	if err != nil {
		return fmt.Errorf("read file %s: %w", i.LocalstackPluginFile, err)
	}
	allinoneplugin := &pluginsv1beta1.Plugin{}
	if err := yaml.Unmarshal(content, allinoneplugin); err != nil {
		return fmt.Errorf("unmarshal file %s: %w", i.LocalstackPluginFile, err)
	}

	// set ns to install ns
	allinoneplugin.Namespace = i.InstallNamespace
	// merge values
	defaultValus := controllers.UnmarshalValues(allinoneplugin.Spec.Values)
	mergedValues := MergeMaps(defaultValus, i.PluginsValues)
	allinoneplugin.Spec.Values = controllers.MarshalValues(mergedValues)

	if err := kube.Apply(ctx, i.Config, []client.Object{allinoneplugin}, kube.WithCreateNamespace()); err != nil {
		return err
	}
	return nil
}

// valus in b overrides in a
func MergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = MergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
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
