package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
)

type Plugin struct {
	Kind       pluginsv1beta1.PluginKind `json:"kind,omitempty"`
	Name       string                    `json:"name,omitempty"`
	Namespace  string                    `json:"namespace,omitempty"`
	Repo       string                    `json:"repo,omitempty"`
	Version    string                    `json:"version,omitempty"`
	Path       string                    `json:"path,omitempty"`
	DryRun     bool                      `json:"dryRun,omitempty"`
	Resources  []runtime.RawExtension    `json:"resources,omitempty"`
	Values     map[string]interface{}    `json:"values,omitempty"`
	FullValues map[string]interface{}    `json:"fullValues,omitempty"`
}

type PluginStatus struct {
	Name              string                       `json:"name,omitempty"`
	Namespace         string                       `json:"namespace,omitempty"`
	Phase             pluginsv1beta1.PluginPhase   `json:"phase,omitempty"`
	Values            map[string]interface{}       `json:"values,omitempty"`
	Version           string                       `json:"version,omitempty"`
	Message           string                       `json:"message,omitempty"`
	Notes             string                       `json:"notes,omitempty"`
	Resources         []*unstructured.Unstructured `json:"resources,omitempty"`
	CreationTimestamp metav1.Time                  `json:"creationTimestamp,omitempty"`
	UpgradeTimestamp  metav1.Time                  `json:"upgradeTimestamp,omitempty"`
	DeletionTimestamp metav1.Time                  `json:"deletionTimestamp,omitempty"`
}

func (s PluginStatus) toPluginStatus() pluginsv1beta1.PluginStatus {
	return pluginsv1beta1.PluginStatus{
		Phase:             s.Phase,
		Message:           s.Message,
		Notes:             s.Notes,
		InstallNamespace:  s.Namespace,
		Values:            pluginsv1beta1.Values{Object: s.Values},
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
		Values:            plugin.Status.Values.Object,
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

func PluginFromPlugin(plugin *pluginsv1beta1.Plugin) *Plugin {
	return &Plugin{
		Name:      plugin.Name,
		Kind:      plugin.Spec.Kind,
		Values:    plugin.Spec.Values.Object,
		Version:   plugin.Spec.Version,
		Repo:      plugin.Spec.Repo,
		Path:      plugin.Spec.Path,
		Resources: plugin.Spec.Resources,
		Namespace: func() string {
			if plugin.Spec.InstallNamespace == "" {
				return plugin.Namespace
			}
			return plugin.Spec.InstallNamespace
		}(),
	}
}
