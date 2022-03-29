package controllers

import (
	"context"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
)

var ErrUnknownPluginKind = errors.New("unknown plugin kind")

type Applier interface {
	// plugin is the plugin to apply,if plugin.path set use it directly.
	Apply(ctx context.Context, plugin Plugin, status *PluginStatus) error
	Remove(ctx context.Context, plugin Plugin, status *PluginStatus) error
}

type PluginOptions struct {
	ChartsDir  string `json:"chartsDir,omitempty"`
	PluginsDir string `json:"pluginsDir,omitempty"`
}

func NewPluginManager(restconfig *rest.Config, options *PluginOptions) *PluginManager {
	return &PluginManager{
		appliers: map[pluginsv1beta1.PluginKind]Applier{
			pluginsv1beta1.PluginKindHelm:   NewHelmApplier(restconfig, options.ChartsDir),
			pluginsv1beta1.PluginKindNative: NewNativeApplier(restconfig, options.PluginsDir),
		},
	}
}

func (m *PluginManager) Apply(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	applier, ok := m.appliers[plugin.Kind]
	if !ok {
		return ErrUnknownPluginKind
	}
	return applier.Apply(ctx, plugin, status)
}

func (m *PluginManager) Remove(ctx context.Context, plugin Plugin, status *PluginStatus) error {
	applier, ok := m.appliers[plugin.Kind]
	if !ok {
		return ErrUnknownPluginKind
	}
	return applier.Remove(ctx, plugin, status)
}

type PluginManager struct {
	appliers map[pluginsv1beta1.PluginKind]Applier
}

type Plugin struct {
	Kind      pluginsv1beta1.PluginKind `json:"kind,omitempty"`
	Name      string                    `json:"name,omitempty"`
	Namespace string                    `json:"namespace,omitempty"`
	Repo      string                    `json:"repo,omitempty"`
	Version   string                    `json:"version,omitempty"`
	Path      string                    `json:"path,omitempty"`
	Values    map[string]interface{}    `json:"values,omitempty"`
}

type PluginStatus struct {
	Name              string                     `json:"name,omitempty"`
	Namespace         string                     `json:"namespace,omitempty"`
	Phase             pluginsv1beta1.PluginPhase `json:"phase,omitempty"`
	Values            map[string]interface{}     `json:"values,omitempty"`
	Version           string                     `json:"version,omitempty"`
	Message           string                     `json:"message,omitempty"`
	Notes             string                     `json:"notes,omitempty"`
	CreationTimestamp metav1.Time                `json:"creationTimestamp,omitempty"`
	UpgradeTimestamp  metav1.Time                `json:"upgradeTimestamp,omitempty"`
	DeletionTimestamp metav1.Time                `json:"deletionTimestamp,omitempty"`
}

func (s PluginStatus) toPluginStatus() pluginsv1beta1.PluginStatus {
	return pluginsv1beta1.PluginStatus{
		Phase:             s.Phase,
		Message:           s.Message,
		Notes:             s.Notes,
		InstallNamespace:  s.Namespace,
		Values:            MarshalValues(s.Values),
		Version:           s.Version,
		CreationTimestamp: s.CreationTimestamp,
		UpgradeTimestamp:  s.UpgradeTimestamp,
		DeletionTimestamp: func() *metav1.Time {
			if s.DeletionTimestamp.IsZero() {
				return nil
			}
			return &s.DeletionTimestamp
		}(),
	}
}

func PluginStatusFromPlugin(plugin *pluginsv1beta1.Plugin) *PluginStatus {
	if plugin == nil {
		return nil
	}
	return &PluginStatus{
		Name:              plugin.Name,
		Namespace:         plugin.Status.InstallNamespace,
		Phase:             plugin.Status.Phase,
		Message:           plugin.Status.Message,
		Values:            UnmarshalValues(plugin.Status.Values),
		Version:           plugin.Status.Version,
		Notes:             plugin.Status.Notes,
		CreationTimestamp: plugin.CreationTimestamp,
		UpgradeTimestamp:  plugin.Status.UpgradeTimestamp,
		DeletionTimestamp: func() metav1.Time {
			if plugin.DeletionTimestamp.IsZero() {
				return metav1.Time{}
			}
			return *plugin.DeletionTimestamp
		}(),
	}
}

func PluginFromPlugin(plugin *pluginsv1beta1.Plugin) Plugin {
	return Plugin{
		Name:    plugin.Name,
		Kind:    plugin.Spec.Kind,
		Values:  UnmarshalValues(plugin.Spec.Values),
		Version: plugin.Spec.Version,
		Repo:    plugin.Spec.Repo,
		Path:    plugin.Spec.Path,
		Namespace: func() string {
			if plugin.Spec.InstallNamespace == "" {
				return plugin.Namespace
			}
			return plugin.Spec.InstallNamespace
		}(),
	}
}
