package gemsplugin

import (
	"context"
	"encoding/json"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"kubegems.io/pkg/agent/cluster"
	pluginscommon "kubegems.io/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TypeCorePlugins       = "core"
	TypeKubernetesPlugins = "kubernetes"
)

type Plugins struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec struct {
		ClusterName       string             `json:"cluster_name"`
		Runtime           string             `json:"runtime"`
		Global            interface{}        `json:"global"`
		CorePlugins       map[string]*Plugin `json:"core_plugins"`
		KubernetesPlugins map[string]*Plugin `json:"kubernetes_plugins"`
	} `json:"spec"`

	Status interface{} `json:"status"`
}

type Plugin struct {
	Name         string `json:"name,omitempty"` // 返回给前端
	Enabled      bool   `json:"enabled"`
	Namespace    string `json:"namespace"`
	Details      `json:"details"`
	Status       `json:"status"`
	Type         string      `json:"-"` // 用于暂存类型给prometheus
	Operator     interface{} `json:"operator,omitempty"`
	Manual       interface{} `json:"manual,omitempty"`
	DefaultClass bool        `json:"default_class,omitempty"`
}

type Details struct {
	Description string `json:"description"`
	Catalog     string `json:"catalog"`
	Version     string `json:"version"`
}

type Status struct {
	Required    bool     `json:"required,omitempty"`
	Deployment  []string `json:"deployment,omitempty"`
	Statefulset []string `json:"statefulset,omitempty"`
	Daemonset   []string `json:"daemonset,omitempty"`
	IsHealthy   bool     `json:"healthy,omitempty"` // 返回给前端
	Host        string   `json:"host,omitempty"`
}

type PluginState struct {
	Annotations map[string]string      `json:"annotations"`
	Enabled     bool                   `json:"enabled"`
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Values      map[string]interface{} `json:"values"`
}

func ListPlugins(ctx context.Context, cli client.Client) ([]PluginState, error) {
	allinoneplugin := &pluginsv1beta1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginscommon.KubeGemsLocalPluginsName,
			Namespace: pluginscommon.KubeGemsLocalPluginsNamespace,
		},
	}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(allinoneplugin), allinoneplugin); err != nil {
		return nil, err
	}
	vals := allinoneplugin.Spec.Values.Object
	_ = vals

	globalValues := vals["global"]
	_ = globalValues
	delete(vals, "global")

	var plugins map[string]PluginState
	if err := json.Unmarshal(allinoneplugin.Spec.Values.Raw, &plugins); err != nil {
		return nil, err
	}
	result := []PluginState{}
	for name, plugin := range plugins {
		if name == "global" {
			continue
		}
		plugin.Name = name
		result = append(result, plugin)
	}
	return result, nil
}

func EnablePlugin(ctx context.Context, cli client.Client, name string, enable bool) error {
	allinoneplugin := &pluginsv1beta1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginscommon.KubeGemsLocalPluginsName,
			Namespace: pluginscommon.KubeGemsLocalPluginsNamespace,
		},
	}
	patch := client.RawPatch(types.MergePatchType, []byte(`{"spec":{"values":{"`+name+`":{"enabled":`+strconv.FormatBool(enable)+`}}}}`))
	return cli.Patch(ctx, allinoneplugin, patch)
}

func GetPlugins(dis discovery.DiscoveryInterface) (*Plugins, error) {
	gemsplugins := &Plugins{}
	// TODO: remove this function
	return gemsplugins, nil
}

func IsPluginHelthy(clus cluster.Interface, plugin *Plugin) bool {
	if !plugin.Enabled {
		return false
	}
	// TODO: plgin status
	return true
}
