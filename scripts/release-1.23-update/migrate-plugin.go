// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"log"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	pluginscommon "kubegems.io/kubegems/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/utils/harbor"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	OfficalChartsRepository = "https://charts.kubegems.io/kubegems"
	DefaultPluginVersion    = "1.0.0"
)

// MigratePlugins migrate plugin cr from kubegems and kubegems-local namespace to kubegems-installer namespace
func MigratePlugins(ctx context.Context, cli client.Client, kubegemsVersion string) error {
	log.Print("migrating kubegems plugins")
	fromns := []string{"kubegems", "kubegems-local"}
	if err := preCheck(ctx, cli); err != nil {
		return err
	}
	for _, nsFrom := range fromns {
		if err := migrateConfigmaps(ctx, cli, nsFrom, kubegemsVersion); err != nil {
			return err
		}
		if err := migratePlugins(ctx, cli, nsFrom, kubegemsVersion); err != nil {
			return err
		}
		if err := cleanOldplugins(ctx, cli, nsFrom); err != nil {
			return err
		}
	}
	// use: kubectl apply -f https://github.com/kubegems/kubegems/raw/release-1.23/deploy/installer.yaml
	// return scaleUpInstaller(ctx, cli, kubegemsVersion)
	return nil
}

func preCheck(ctx context.Context, cli client.Client) error {
	// scale installer replicas to 0
	patch0 := client.RawPatch(types.MergePatchType, []byte(`{"spec":{"replicas":0}}`))
	depinstaller := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubegems-installer",
			Namespace: pluginscommon.KubeGemsNamespaceInstaller,
		},
	}
	log.Printf("scale deployment %s/%s to 0 ", depinstaller.Name, depinstaller.Namespace)
	if err := cli.Patch(ctx, depinstaller, patch0); err != nil {
		return err
	}

	// wait pod termination
	for {
		time.Sleep(time.Second)
		podlist := &corev1.PodList{}
		err := cli.List(ctx, podlist, client.InNamespace(depinstaller.Namespace))
		if err != nil {
			return err
		}
		if count := len(podlist.Items); count != 0 {
			log.Printf("wait %d pod terminating in ns %s", count, depinstaller.Namespace)
		} else {
			log.Printf("all pod terminating in ns %s", depinstaller.Namespace)
			return nil
		}
	}
}

func migrateConfigmaps(ctx context.Context, cli client.Client, fromns string, kubegemsVersion string) error {
	log.Printf("migrating configmaps in namespace %s", fromns)

	cmlist := &corev1.ConfigMapList{}
	if err := cli.List(ctx, cmlist, client.InNamespace(fromns)); err != nil {
		return err
	}
	for _, val := range cmlist.Items {
		if !strings.HasSuffix(val.Name, "-values") {
			continue
		}
		log.Printf("migrate configmap [%s]", val.Name)
		if val.Name == "kubegems-global-values" {
			// rename keys
			newdata := map[string]string{}
			for k, v := range val.Data {
				k = strings.TrimPrefix(k, "global.")
				newdata[k] = v
			}
			if kubegemsVersion != "" {
				// override kubegems version
				newdata["kubegemsVersion"] = kubegemsVersion
			}
			val.Data = newdata
			// create global plugin
			globalplugin := &pluginsv1beta1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pluginscommon.KubegemsChartGlobal,
					Namespace: pluginscommon.KubeGemsNamespaceInstaller,
				},
			}
			log.Printf("create/update global plugin")
			if _, err := controllerutil.CreateOrUpdate(ctx, cli, globalplugin, func() error {
				return updateGlobalValueFromCM(&val, globalplugin)
			}); err != nil {
				return err
			}
			continue
		}
		val.Namespace = pluginscommon.KubeGemsNamespaceInstaller
		val.ResourceVersion = ""
		tmp := val.DeepCopy()
		log.Printf("migrate values for configmap [%s]", tmp.Name)
		// move to installer ns
		if _, err := controllerutil.CreateOrUpdate(ctx, cli, tmp, func() error {
			tmp.Data = val.Data
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func updateGlobalValueFromCM(cm *corev1.ConfigMap, global *pluginsv1beta1.Plugin) error {
	if global.Annotations == nil {
		global.Annotations = map[string]string{}
	}

	pluginvalues := map[string]any{}
	for k, v := range cm.Data {
		pluginvalues[k] = v
	}

	global.Annotations[pluginscommon.AnnotationCategory] = "core/KubeGems"
	// create global plugin
	global.Spec = pluginsv1beta1.PluginSpec{
		URL:     OfficalChartsRepository,
		Kind:    pluginsv1beta1.BundleKindTemplate,
		Version: DefaultPluginVersion,
		Values:  pluginsv1beta1.Values{Object: pluginvalues},
	}
	global.Status = pluginsv1beta1.PluginStatus{}
	return nil
}

func migratePlugins(ctx context.Context, cli client.Client, fromns string, kubegemsVersion string) error {
	log.Printf("migrating plugins in namespace %s", fromns)

	pluginlist := &pluginsv1beta1.PluginList{}
	if err := cli.List(ctx, pluginlist, client.InNamespace(fromns)); err != nil {
		return err
	}

	for _, plugin := range pluginlist.Items {
		if plugin.Spec.Disabled {
			log.Printf("skip disabled plugin [%s]", plugin.Name)
			continue
		}
		log.Printf("migrate plugin [%s]", plugin.Name)

		newplugin := &pluginsv1beta1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      plugin.Name,
				Namespace: pluginscommon.KubeGemsNamespaceInstaller,
			},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, cli, newplugin, func() error {
			return migratePlugin(newplugin, &plugin, kubegemsVersion)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func migratePlugin(plugin *pluginsv1beta1.Plugin, old *pluginsv1beta1.Plugin, newkubegemsChartVersion string) error {
	plugin.Finalizers = old.Finalizers
	// annotations
	plugin.Annotations = old.Annotations
	delete(plugin.Annotations, "plugins.kubegems.io/appVersion")
	maincate, ok := plugin.Annotations["plugins.kubegems.io/main-category"]
	if ok {
		delete(plugin.Annotations, "plugins.kubegems.io/main-category")
		cate := plugin.Annotations[pluginscommon.AnnotationCategory]
		if !strings.Contains(cate, "/") {
			plugin.Annotations[pluginscommon.AnnotationCategory] = maincate + "/" + cate
		}
	}

	// spec
	plugin.Spec = old.Spec
	if old.Name == "kubegems" {
		if plugin.Annotations == nil {
			plugin.Annotations = map[string]string{}
		}
		plugin.Annotations[pluginscommon.AnnotationCategory] = "core/Kubegems"
		plugin.Spec.InstallNamespace = "kubegems"
		plugin.Spec.ValuesFrom = []pluginsv1beta1.ValuesFrom{
			{
				Kind:   "ConfigMap",
				Name:   "kubegems-global-values",
				Prefix: "global.",
			},
		}
		plugin.Spec.Values.Raw = nil
		delete(plugin.Spec.Values.Object, "global")
	}
	plugin.Spec.URL = "https://charts.kubegems.io/kubegems"
	if strings.HasPrefix(plugin.Name, "kubegems") {
		plugin.Spec.Version = strings.TrimPrefix(newkubegemsChartVersion, "v") // latest kubegems version
	} else {
		plugin.Spec.Version = DefaultPluginVersion // default plugin version
	}

	for i, from := range plugin.Spec.ValuesFrom {
		if from.Name == "kubegems-global-values" {
			plugin.Spec.ValuesFrom[i].Prefix = "global."
		}
	}
	return nil
}

func cleanOldplugins(ctx context.Context, cli client.Client, fromns string) error {
	log.Printf("clean all old plugins in namespace %s", fromns)
	pluginlist := &pluginsv1beta1.PluginList{}
	if err := cli.List(ctx, pluginlist, client.InNamespace(fromns)); err != nil {
		return err
	}
	for _, val := range pluginlist.Items {
		val.Finalizers = nil
		if err := cli.Update(ctx, &val); err != nil {
			return err
		}
	}
	return cli.DeleteAllOf(ctx, &pluginsv1beta1.Plugin{}, client.InNamespace(fromns))
}

func scaleUpInstaller(ctx context.Context, cli client.Client, kubegemsVersion string) error {
	depinstaller := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubegems-installer",
			Namespace: pluginscommon.KubeGemsNamespaceInstaller,
		},
	}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(depinstaller), depinstaller); err != nil {
		return err
	}

	patch := client.MergeFrom(depinstaller.DeepCopy())
	for i, container := range depinstaller.Spec.Template.Spec.Containers {
		if !strings.Contains(container.Name, "installer") {
			continue
		}
		// update image tag
		image, _ := harbor.SplitImageNameTag(container.Image)
		fullimage := image + ":" + kubegemsVersion
		log.Printf("scale deployment %s/%s to 1 and set images to %s", depinstaller.Name, depinstaller.Namespace, fullimage)
		depinstaller.Spec.Template.Spec.Containers[i].Image = fullimage
	}
	// sacle up to 1
	depinstaller.Spec.Replicas = pointer.Int32(1)
	return cli.Patch(ctx, depinstaller, patch)
}
