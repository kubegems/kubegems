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

package pluginmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"kubegems.io/kubegems/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/utils"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Bootstrap struct {
	Config    *rest.Config // target cluster config
	Namespace string       // installer namespace
}

type GlobalValues struct {
	ImageRegistry   string `json:"imageRegistry"`
	ImageRepository string `json:"imageRepository"`
	ClusterName     string `json:"clusterName"`
	StorageClass    string `json:"storageClass"`
	Runtime         string `json:"runtime"`
}

func (m *PluginManager) GetGlobalValues(ctx context.Context) (*GlobalValues, error) {
	globalplugin := &pluginsv1beta1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      plugins.KubegemsChartGlobal,
			Namespace: plugins.KubeGemsNamespaceInstaller,
		},
	}
	if err := m.Client.Get(ctx, client.ObjectKeyFromObject(globalplugin), globalplugin); err != nil {
		return nil, err
	}
	ret := map[string]string{}
	for k, v := range globalplugin.Spec.Values.Object {
		if val, ok := v.(string); ok {
			ret[k] = val
		}
	}
	globalVals := &GlobalValues{}
	json.Unmarshal(globalplugin.Spec.Values.Raw, globalVals)
	return globalVals, nil
}

func (i Bootstrap) Install(ctx context.Context, values GlobalValues) error {
	ns := i.Namespace
	if ns == "" {
		ns = plugins.KubeGemsNamespaceInstaller
	}
	cli, err := kube.NewClient(i.Config)
	if err != nil {
		return err
	}
	// v1.21.X -> 1.21.X , cause helm chart version follow pure semver.
	kubegemsVersion := version.Get().GitVersion

	// apply installer
	installerobjects, err := ParseInstallerObjects(plugins.KubegemsPluginsCachePath, values, kubegemsVersion)
	if err != nil {
		return err
	}
	if err = ApplyInNamespace(ctx, cli, ns, installerobjects...); err != nil {
		return fmt.Errorf("apply installer: %v", err)
	}

	// apply plugins
	pm := &PluginManager{Client: cli}

	globalvals := map[string]interface{}{
		"imageRegistry":   values.ImageRegistry,
		"imageRepository": values.ImageRepository,
		"clusterName":     values.ClusterName,
		"storageClass":    values.StorageClass,
		"runtime":         values.Runtime,
	}
	if err := pm.Install(ctx, plugins.KubegemsChartGlobal, "", globalvals); err != nil {
		return err
	}
	cahrtversion := strings.TrimPrefix(kubegemsVersion, "v")
	if err := pm.Install(ctx, plugins.KubegemsChartInstaller, cahrtversion, nil); err != nil {
		return err
	}
	if err := pm.Install(ctx, plugins.KubegemsChartLocal, cahrtversion, nil); err != nil {
		return err
	}
	return nil
}

func (i Bootstrap) Remove(ctx context.Context) error {
	ns := i.Namespace
	if ns == "" {
		ns = plugins.KubeGemsNamespaceInstaller
	}
	_ = ns
	// do nothing
	return nil
}

func ParseInstallerObjects(path string, values GlobalValues, version string) ([]client.Object, error) {
	objects, err := utils.ReadObjectsFromFile[client.Object](filepath.Join(path, "installer.yaml"))
	if err != nil {
		return nil, err
	}
	if version == "" {
		return objects, nil
	}
	// update image of kubegems container
	for _, obj := range objects {
		switch item := obj.(type) {
		case *appsv1.Deployment:
			for i, container := range item.Spec.Template.Spec.Containers {
				if !strings.Contains(container.Image, "kubegems/kubegems") {
					continue
				}
				containerImage := fmt.Sprintf(
					"%s/%s/kubegems:%s", values.ImageRegistry, values.ImageRepository, version,
				)
				item.Spec.Template.Spec.Containers[i].Image = containerImage
			}
		}
	}
	return objects, nil
}

func ApplyInNamespace(ctx context.Context, cli client.Client, ns string, objects ...client.Object) error {
	log := log.FromContextOrDiscard(ctx)
	// check if namespace exists
	if err := cli.Get(ctx, client.ObjectKey{Name: ns}, &corev1.Namespace{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		log.Info("creating namespace", "namespace", ns)
		// create namespace
		if err = cli.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}); err != nil {
			return fmt.Errorf("create namespace: %v", err)
		}
	}

	// create or patch objects
	creatrOrPatch := func(ctx context.Context, obj client.Object) error {
		log := log.WithValues("gvk", obj.GetObjectKind().GroupVersionKind().String(), "namespace", obj.GetNamespace(), "name", obj.GetName())
		if obj.GetNamespace() == "" {
			obj.SetNamespace(ns)
		}
		log.Info("creating object")
		if err := cli.Create(ctx, obj); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				log.Error(err, "create object")
				return err
			}
			// patch object
			obj.SetManagedFields(nil)
			log.Info("patching object")
			if err := cli.Patch(ctx, obj, client.Apply, client.ForceOwnership, client.FieldOwner("kubegems")); err != nil {
				log.Error(err, "patch object")
				return err
			}
			return nil
		}
		return nil
	}

	for _, obj := range objects {
		if err := creatrOrPatch(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}
