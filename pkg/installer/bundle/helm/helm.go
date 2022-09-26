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

package helm

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/utils"
)

type Apply struct {
	Config *rest.Config
}

func New(config *rest.Config) *Apply {
	return &Apply{Config: config}
}

func (r *Apply) Template(ctx context.Context, bundle *pluginsv1beta1.Plugin, into string) ([]byte, error) {
	rls := r.getPreRelease(bundle)
	applyedRelease, err := r.ApplyChart(ctx, rls.Name, rls.Namespace, into, rls.Config, ApplyOptions{DryRun: true})
	if err != nil {
		return nil, err
	}
	return []byte(applyedRelease.Manifest), nil
}

func (r *Apply) Apply(ctx context.Context, bundle *pluginsv1beta1.Plugin, into string) error {
	rls := r.getPreRelease(bundle)
	applyedRelease, err := r.ApplyChart(ctx, rls.Name, rls.Namespace, into, rls.Config, ApplyOptions{})
	if err != nil {
		return err
	}
	bundle.Status.Resources = parseResource([]byte(applyedRelease.Manifest))
	if applyedRelease.Info.Status != release.StatusDeployed {
		return fmt.Errorf("apply not finished:%s", applyedRelease.Info.Description)
	}
	bundle.Status.Phase = pluginsv1beta1.PhaseInstalled
	bundle.Status.Message = applyedRelease.Info.Notes
	bundle.Status.Namespace = applyedRelease.Namespace
	bundle.Status.CreationTimestamp = convtime(applyedRelease.Info.FirstDeployed.Time)
	bundle.Status.UpgradeTimestamp = convtime(applyedRelease.Info.LastDeployed.Time)
	bundle.Status.Values = pluginsv1beta1.Values{Object: applyedRelease.Config}
	bundle.Status.Version = applyedRelease.Chart.Metadata.Version
	bundle.Status.AppVersion = applyedRelease.Chart.Metadata.AppVersion
	return nil
}

func (r *Apply) Remove(ctx context.Context, bundle *pluginsv1beta1.Plugin) error {
	log := logr.FromContextOrDiscard(ctx)
	if bundle.Status.Phase == pluginsv1beta1.PhaseDisabled {
		log.Info("already removed or not installed")
		return nil
	}
	rls := r.getPreRelease(bundle)
	// uninstall
	removedRelease, err := r.RemoveChart(ctx, rls.Name, rls.Namespace, RemoveOptions{})
	if err != nil {
		return err
	}
	log.Info("removed")
	if removedRelease == nil {
		bundle.Status.Phase = pluginsv1beta1.PhaseDisabled
		bundle.Status.Message = "plugin not install"
		return nil
	}
	bundle.Status.Phase = pluginsv1beta1.PhaseDisabled
	bundle.Status.Message = removedRelease.Info.Description
	return nil
}

func (r Apply) getPreRelease(bundle *pluginsv1beta1.Plugin) *release.Release {
	releaseNamespace := bundle.Spec.InstallNamespace
	if releaseNamespace == "" {
		releaseNamespace = bundle.Namespace
	}
	return &release.Release{Name: bundle.Name, Namespace: releaseNamespace, Config: bundle.Spec.Values.Object}
}

// https://github.com/golang/go/issues/19502
// metav1.Time and time.Time are not comparable directly
func convtime(t time.Time) metav1.Time {
	t, _ = time.Parse(time.RFC3339, t.Format(time.RFC3339))
	return metav1.Time{Time: t}
}

func parseResource(resources []byte) []corev1.ObjectReference {
	ress, _ := utils.SplitYAML(resources)
	managedResources := make([]corev1.ObjectReference, len(ress))
	for i, res := range ress {
		managedResources[i] = utils.GetReference(res)
	}
	return managedResources
}
