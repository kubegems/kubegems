package controllers

import (
	"encoding/json"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"sigs.k8s.io/yaml"
)

type Plugin struct {
	Kind      pluginsv1beta1.PluginKind `json:"kind,omitempty"`
	Name      string                    `json:"name,omitempty"`
	Namespace string                    `json:"namespace,omitempty"`
	Repo      string                    `json:"repo,omitempty"`
	Version   string                    `json:"version,omitempty"`
	Path      string                    `json:"path,omitempty"`
	DryRun    bool                      `json:"dryRun,omitempty"`
	Resources []runtime.RawExtension    `json:"resources,omitempty"`
	Values    map[string]interface{}    `json:"values,omitempty"`
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
		Name:      plugin.Name,
		Kind:      plugin.Spec.Kind,
		Values:    UnmarshalValues(plugin.Spec.Values),
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

func MarshalValues(vals map[string]interface{}) runtime.RawExtension {
	if vals == nil {
		return runtime.RawExtension{}
	}
	bytes, _ := json.Marshal(vals)
	return runtime.RawExtension{Raw: bytes}
}

func UnmarshalValues(val runtime.RawExtension) map[string]interface{} {
	if val.Raw == nil {
		return nil
	}
	var vals interface{}
	_ = yaml.Unmarshal(val.Raw, &vals)

	if kvs, ok := vals.(map[string]interface{}); ok {
		RemoveNulls(kvs)
		return kvs
	}
	if arr, ok := vals.([]interface{}); ok {
		// is format of --set K=V
		kvs := make(map[string]interface{}, len(arr))
		for _, kv := range arr {
			if kv, ok := kv.(map[string]interface{}); ok {
				for k, v := range kv {
					kvs[k] = v
				}
			}
		}
		RemoveNulls(kvs)
		return kvs
	}
	return nil
}

// https://github.com/helm/helm/blob/bed1a42a398b30a63a279d68cc7319ceb4618ec3/pkg/chartutil/coalesce.go#L37
// helm CoalesceValues cant handle nested null,like `{a: {b: null}}`, which want to be `{}`
func RemoveNulls(m map[string]interface{}) {
	val := reflect.ValueOf(m)
	for _, e := range val.MapKeys() {
		v := val.MapIndex(e)
		switch t := v.Interface().(type) {
		case map[string]interface{}:
			RemoveNulls(t)
		}

		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		if (v.Kind() == reflect.Invalid) || (v.Kind() == reflect.Map && v.Len() == 0) || v.IsZero() {
			delete(m, e.String())
			continue
		}
	}
}
