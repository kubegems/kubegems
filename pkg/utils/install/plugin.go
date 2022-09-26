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

package install

import (
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pluginscommon "kubegems.io/kubegems/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ParsePluginsFrom(path string, values GlobalValues) ([]client.Object, error) {
	globalvalues := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: GlobalValuesConfigMapName,
		},
		Data: map[string]string{
			"global.imageRegistry":   values.ImageRegistry,
			"global.imageRepository": values.ImageRepository,
			"global.clusterName":     values.ClusterName,
			"global.storageClass":    values.StorageClass,
			"global.kubegemsVersion": values.KubegemsVersion,
			"global.runtime":         values.Runtime,
		},
	}
	allobjects := []client.Object{
		globalvalues, // global configmap
	}
	addplugin := func(dir string) {
		//  read plugin.yaml in dir
		plugins, _ := utils.ReadObjectsFromFile[*pluginsv1beta1.Plugin](
			filepath.Join(dir, "plugin.yaml"),
		)

		//  read Chat.yaml
		if chartmeta, _ := chartutil.LoadChartfile(filepath.Join(dir, "Chart.yaml")); chartmeta != nil {
			for _, item := range plugins {
				if item.Annotations == nil {
					item.Annotations = make(map[string]string)
				}
				item.Annotations[pluginscommon.AnnotationDescription] = chartmeta.Description
				item.Annotations[pluginscommon.AnnotationAppVersion] = chartmeta.AppVersion
			}
		}
		// add global values to plugin
		for _, item := range plugins {
			injectGlobalValues(item)
			allobjects = append(allobjects, item)
		}
	}
	forDir(path, addplugin)
	return allobjects, nil
}

func injectGlobalValues(plugin *pluginsv1beta1.Plugin) {
	globalref := pluginsv1beta1.ValuesFrom{
		Name: GlobalValuesConfigMapName,
		Kind: "ConfigMap",
	}
	for _, ref := range plugin.Spec.ValuesFrom {
		if ref == globalref {
			return
		}
	}
	// inject prepend
	plugin.Spec.ValuesFrom = append([]pluginsv1beta1.ValuesFrom{globalref}, plugin.Spec.ValuesFrom...)
}

func forDir(root string, f func(path string)) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		f(path)
	}
	return nil
}
