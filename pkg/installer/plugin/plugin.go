package plugin

import (
	"context"
	"fmt"

	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PluginApplier struct {
	helm    *HelmApplier
	Options *Options
	native  *NativeApplier
}

func NewApplier(cfg *rest.Config, cli client.Client, options *Options) *PluginApplier {
	return &PluginApplier{
		helm:    NewHelm(cfg, options.Cache),
		native:  NewNative(cfg, cli),
		Options: options,
	}
}

func (b *PluginApplier) Template(ctx context.Context, bundle *pluginsv1beta1.Plugin) ([]byte, error) {
	into, err := Download(ctx, bundle, b.Options.Cache, b.Options.SearchDirs...)
	if err != nil {
		return nil, err
	}
	switch bundle.Spec.Kind {
	case pluginsv1beta1.PluginKindHelm:
		return b.helm.Template(ctx, bundle, into)
	case pluginsv1beta1.PluginKindKustomize, pluginsv1beta1.PluginKindTemplate:
		return b.native.Template(ctx, bundle, into)
	default:
		return nil, fmt.Errorf("unknown bundle kind: %s", bundle.Spec.Kind)
	}
}

func (b *PluginApplier) Apply(ctx context.Context, bundle *pluginsv1beta1.Plugin) error {
	into, err := Download(ctx, bundle, b.Options.Cache, b.Options.SearchDirs...)
	if err != nil {
		return err
	}
	switch bundle.Spec.Kind {
	case pluginsv1beta1.PluginKindHelm:
		return b.helm.Apply(ctx, bundle, into)
	case pluginsv1beta1.PluginKindKustomize, pluginsv1beta1.PluginKindTemplate:
		return b.native.Apply(ctx, bundle, into)
	default:
		return fmt.Errorf("unknown bundle kind: %s", bundle.Spec.Kind)
	}
}

func (b *PluginApplier) Remove(ctx context.Context, bundle *pluginsv1beta1.Plugin) error {
	switch bundle.Spec.Kind {
	case pluginsv1beta1.PluginKindHelm:
		return b.helm.Remove(ctx, bundle)
	case pluginsv1beta1.PluginKindKustomize, pluginsv1beta1.PluginKindTemplate:
		return b.native.Remove(ctx, bundle)
	default:
		return fmt.Errorf("unknown bundle kind: %s", bundle.Spec.Kind)
	}
}
