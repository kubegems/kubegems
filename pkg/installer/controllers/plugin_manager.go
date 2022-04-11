package controllers

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
)

var ErrUnknownPluginKind = errors.New("unknown plugin kind")

type PluginInstaller interface {
	// plugin is the plugin to apply,if plugin.path set use it directly.
	Template(ctx context.Context, plugin Plugin) ([]byte, error)
	Apply(ctx context.Context, plugin Plugin, status *PluginStatus) error
	Remove(ctx context.Context, plugin Plugin, status *PluginStatus) error
}

type PluginOptions struct {
	PluginsDir string `json:"pluginsDir,omitempty"`
}

func NewPluginManager(restconfig *rest.Config, options *PluginOptions) *PluginManager {
	return &PluginManager{
		Installers: map[pluginsv1beta1.PluginKind]PluginInstaller{
			pluginsv1beta1.PluginKindHelm:      NewHelmPlugin(restconfig, options.PluginsDir),
			pluginsv1beta1.PluginKindKustomize: NewNativePlugin(restconfig, options.PluginsDir, KustomizeTemplatePlugin),
			pluginsv1beta1.PluginKindTemplate:  NewNativePlugin(restconfig, options.PluginsDir, TemplatesTemplatePlugin),
			pluginsv1beta1.PluginKindInline:    NewNativePlugin(restconfig, options.PluginsDir, InlineTemplatePlugin),
		},
		Options: options,
	}
}

type PluginManagerOptions struct {
	DryRun bool
}

type PluginManagerOption func(*PluginManagerOptions)

func WithDryRun() PluginManagerOption {
	return func(options *PluginManagerOptions) {
		options.DryRun = true
	}
}

func (m *PluginManager) Template(ctx context.Context, apiplugin *pluginsv1beta1.Plugin) ([]byte, error) {
	thisPlugin := PluginFromPlugin(apiplugin)
	thisPlugin.DryRun = true // must set this

	installer, ok := m.Installers[thisPlugin.Kind]
	if !ok {
		return nil, ErrUnknownPluginKind
	}
	return installer.Template(ctx, thisPlugin)
}

func (m *PluginManager) Download(ctx context.Context, apiplugin *pluginsv1beta1.Plugin) error {
	plugin := PluginFromPlugin(apiplugin)
	return DownloadPlugin(ctx, &plugin, m.Options.PluginsDir)
}

func (m *PluginManager) Apply(ctx context.Context, apiplugin *pluginsv1beta1.Plugin, options ...PluginManagerOption) error {
	thisPlugin := PluginFromPlugin(apiplugin)
	thisStatus := PluginStatusFromPlugin(apiplugin)

	opts := &PluginManagerOptions{}
	for _, o := range options {
		o(opts)
	}

	if opts.DryRun {
		thisPlugin.DryRun = true
	}

	err := m.apply(ctx, thisPlugin, thisStatus)
	apiplugin.Status = thisStatus.toPluginStatus()
	return err
}

func (m *PluginManager) apply(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	installer, ok := m.Installers[plugin.Kind]
	if !ok {
		return ErrUnknownPluginKind
	}
	if err := installer.Apply(ctx, plugin, status); err != nil {
		status.Phase = pluginsv1beta1.PluginPhaseFailed
		status.Message = err.Error()
		return err
	}
	return nil
}

func (m *PluginManager) Remove(ctx context.Context, apiplugin *pluginsv1beta1.Plugin) error {
	thisPlugin := PluginFromPlugin(apiplugin)
	thisStatus := PluginStatusFromPlugin(apiplugin)

	applier, ok := m.Installers[thisPlugin.Kind]
	if !ok {
		return ErrUnknownPluginKind
	}
	if err := applier.Remove(ctx, thisPlugin, thisStatus); err != nil {
		apiplugin.Status = thisStatus.toPluginStatus()
		apiplugin.Status.Phase = pluginsv1beta1.PluginPhaseFailed
		apiplugin.Status.Message = err.Error()
		return err
	}
	apiplugin.Status = thisStatus.toPluginStatus()
	return nil
}

type PluginManager struct {
	Installers map[pluginsv1beta1.PluginKind]PluginInstaller
	Options    *PluginOptions
}

func DetectPluginType(path string) pluginsv1beta1.PluginKind {
	// helm ?
	if _, err := os.Stat(filepath.Join(path, "Chart.yaml")); err == nil {
		return pluginsv1beta1.PluginKindHelm
	}
	// kustomize ?
	if _, err := os.Stat(filepath.Join(path, "kustomization.yaml")); err == nil {
		return pluginsv1beta1.PluginKindKustomize
	}
	// default template
	return pluginsv1beta1.PluginKindTemplate
}
