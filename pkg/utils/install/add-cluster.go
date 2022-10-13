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
	"context"
	"fmt"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	bundlev1 "kubegems.io/bundle-controller/pkg/apis/bundle/v1beta1"
	"kubegems.io/bundle-controller/pkg/utils"
	pluginscommon "kubegems.io/kubegems/pkg/apis/plugins"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	KubeGemPluginsPath            = "plugins"
	GlobalValuesConfigMapName     = pluginscommon.KubeGemsGlobalValuesConfigMapName
	KubeGemsLocalPluginsNamespace = pluginscommon.KubeGemsLocalPluginsNamespace
	KubeGemsInstallerNamespace    = "kubegems-installer"
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
	Runtime         string `json:"runtime"`
}

func (i OpratorInstaller) Apply(ctx context.Context, ns string, values GlobalValues) error {
	if ns == "" {
		ns = KubeGemsLocalPluginsNamespace
	}
	cli, err := kube.NewClient(i.Config)
	if err != nil {
		return err
	}

	// apply installer.yaml
	installerobjects, err := ParseInstallerFrom(KubeGemPluginsPath, values)
	if err != nil {
		return err
	}
	if err = CreateOrPatchInNamespace(ctx, cli, KubeGemsInstallerNamespace, installerobjects...); err != nil {
		return fmt.Errorf("apply installer: %v", err)
	}

	// apply local plugins.yaml
	plugins, err := ParsePluginsFrom(KubeGemPluginsPath, values)
	if err != nil {
		return err
	}
	if err = CreateOrPatchInNamespace(ctx, cli, ns, plugins...); err != nil {
		return fmt.Errorf("apply local plugins: %v", err)
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
	// remove all bundles in ns
	log.FromContextOrDiscard(ctx).Info("removing kubegems plugins", "namespace", ns)
	return cli.DeleteAllOf(ctx, &bundlev1.Bundle{}, client.InNamespace(ns))
	// remove kubegems-installer by hand
}

func ParseInstallerFrom(path string, values GlobalValues) ([]client.Object, error) {
	objects, err := utils.ReadObjectsFromFile[client.Object](filepath.Join(path, "installer.yaml"))
	if err != nil {
		return nil, err
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
					"%s/%s/kubegems:%s", values.ImageRegistry, values.ImageRepository, values.KubegemsVersion,
				)
				item.Spec.Template.Spec.Containers[i].Image = containerImage
			}
		}
	}
	return objects, nil
}

func CreateOrPatchInNamespace(ctx context.Context, cli client.Client, ns string, objects ...client.Object) error {
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
