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

package bundle

import (
	"context"
	"fmt"

	"k8s.io/client-go/rest"
	plugins "kubegems.io/kubegems/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/bundle/helm"
	"kubegems.io/kubegems/pkg/installer/bundle/kustomize"
	"kubegems.io/kubegems/pkg/installer/bundle/native"
	"kubegems.io/kubegems/pkg/installer/bundle/template"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Apply interface {
	Apply(ctx context.Context, bundle *pluginsv1beta1.Plugin, into string) error
	Remove(ctx context.Context, bundle *pluginsv1beta1.Plugin) error
	Template(ctx context.Context, bundle *pluginsv1beta1.Plugin, into string) ([]byte, error)
}

type BundleApplier struct {
	Options  *Options
	appliers map[pluginsv1beta1.BundleKind]Apply
}

type Options struct {
	CacheDir string
}

func NewDefaultOptions() *Options {
	return &Options{CacheDir: plugins.KubegemsPluginsCachePath}
}

func NewDefaultApply(cfg *rest.Config, cli client.Client, options *Options) *BundleApplier {
	return &BundleApplier{
		Options: options,
		appliers: map[pluginsv1beta1.BundleKind]Apply{
			pluginsv1beta1.BundleKindHelm:      helm.New(cfg),
			pluginsv1beta1.BundleKindKustomize: native.New(cli, kustomize.KustomizeBuildFunc),
			pluginsv1beta1.BundleKindTemplate:  native.New(cli, template.NewTemplaterFunc(cfg)),
			pluginsv1beta1.BundleKindNative:    native.New(cli, native.NewNativeFunc),
		},
	}
}

func (b *BundleApplier) Template(ctx context.Context, bundle *pluginsv1beta1.Plugin) ([]byte, error) {
	into, err := b.Download(ctx, bundle)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	if apply, ok := b.appliers[bundle.Spec.Kind]; ok {
		return apply.Template(ctx, bundle, into)
	}
	return nil, fmt.Errorf("unknown bundle kind: %s", bundle.Spec.Kind)
}

func (b *BundleApplier) Download(ctx context.Context, bundle *pluginsv1beta1.Plugin) (string, error) {
	name := bundle.Name
	if chart := bundle.Spec.Chart; chart != "" {
		name = chart
	}
	version := bundle.Spec.Version
	if version == "" {
		version = bundle.Status.Version // use the installed version
	}
	return Download(ctx,
		bundle.Spec.URL,
		name,
		version,
		bundle.Spec.Path,
		b.Options.CacheDir,
	)
}

func (b *BundleApplier) Apply(ctx context.Context, bundle *pluginsv1beta1.Plugin) error {
	into, err := b.Download(ctx, bundle)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	if apply, ok := b.appliers[bundle.Spec.Kind]; ok {
		return apply.Apply(ctx, bundle, into)
	}
	return fmt.Errorf("unknown bundle kind: %s", bundle.Spec.Kind)
}

func (b *BundleApplier) Remove(ctx context.Context, bundle *pluginsv1beta1.Plugin) error {
	if apply, ok := b.appliers[bundle.Spec.Kind]; ok {
		return apply.Remove(ctx, bundle)
	}
	return fmt.Errorf("unknown bundle kind: %s", bundle.Spec.Kind)
}
