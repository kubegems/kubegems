package clusterhandler

import (
	"context"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	pluginscommon "kubegems.io/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"kubegems.io/pkg/installer/helm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	KubeGemPluginsPath               = "plugins"
	KubeGemsInstallerPluginName      = pluginscommon.KubeGemsInstallerPluginsNamespace
	KubeGemsInstallerPluginNamespace = pluginscommon.KubeGemsInstallerPluginsNamespace
	KubeGemsLocalPluginsNamespace    = pluginscommon.KubeGemsLocalPluginsNamespace
	KubeGemsLocalPluginsName         = pluginscommon.KubeGemsLocalPluginsName
)

type OpratorInstaller struct {
	Config *rest.Config // target cluster config
}

type GlobalValues struct {
	ImageRegistry   string `json:"imageRegistry"`
	ImageRepository string `json:"imageRepository"`
	ClusterName     string `json:"clusterName"`
	StorageClass    string `json:"storageClass"`
	KubegemsVersion string `json:"kubegemsVersion"`
}

func (i OpratorInstaller) Apply(ctx context.Context, ns string, values GlobalValues) error {
	if ns == "" {
		ns = KubeGemsLocalPluginsNamespace
	}
	chartpath := filepath.Join(KubeGemPluginsPath, KubeGemsInstallerPluginName)
	log.FromContextOrDiscard(ctx).Info("applying kubegems-installer chart", "chartPath", chartpath)
	// 	installer:
	//   image:
	//     registry: docker.io
	//     repository: kubegems/kubegems
	//     tag: latest
	installerValues := map[string]interface{}{
		"installer": map[string]interface{}{
			"image": map[string]interface{}{
				"registry": values.ImageRegistry,
				"tag":      values.KubegemsVersion,
			},
		},
	}
	relese, err := (&helm.Helm{Config: i.Config}).ApplyChart(ctx,
		KubeGemsInstallerPluginName, KubeGemsInstallerPluginNamespace,
		chartpath, installerValues, helm.ApplyOptions{},
	)
	if err != nil {
		return err
	}
	log.Info("apply kubegems installer succeed", "namespace", relese.Namespace, "name", relese.Name, "version", relese.Version)
	allinoneplugin := &pluginsv1beta1.Plugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: pluginsv1beta1.GroupVersion.String(),
			Kind:       "Plugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeGemsLocalPluginsName,
			Namespace: ns,
		},
		Spec: pluginsv1beta1.PluginSpec{
			Enabled:          true,
			Kind:             pluginsv1beta1.PluginKindTemplate,
			InstallNamespace: ns, // set ns to install ns
			Values: pluginsv1beta1.Values{Object: map[string]interface{}{
				"global": map[string]interface{}{
					"imageRegistry":   values.ImageRegistry,   // eg. docker.io or registry.cn-hangzhou.aliyuncs.com
					"imageRepository": values.ImageRepository, // eg. kubegems, kubegems-testing
					"clusterName":     values.ClusterName,
					"storageClass":    values.StorageClass,
				},
				"kubegems-installer": map[string]interface{}{
					"version": values.KubegemsVersion,
				},
				"kubegems-local": map[string]interface{}{
					"version": values.KubegemsVersion,
				},
			}},
		},
	}
	log.Info("applying kubegems plugins", "plugin", allinoneplugin)
	if err := kube.Apply(ctx, i.Config, []client.Object{allinoneplugin}, kube.WithCreateNamespace()); err != nil {
		return err
	}
	return nil
}

func (i OpratorInstaller) Remove(ctx context.Context, ns string) error {
	if ns == "" {
		ns = KubeGemsLocalPluginsNamespace
	}
	cli, err := kube.NewClient(i.Config)
	if err != nil {
		return err
	}
	// remove all in one plugin
	allinoneplugin := &pluginsv1beta1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeGemsLocalPluginsName,
			Namespace: ns,
		},
	}
	log.FromContextOrDiscard(ctx).Info("removing kubegems plugins", "plugin", allinoneplugin)
	return cli.Delete(ctx, allinoneplugin)
	// remove kubegems-installer by hand
}
