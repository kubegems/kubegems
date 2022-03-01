package gemsplugin

import (
	"context"
	"encoding/json"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/log"
)

const (
	TypeCorePlugins       = "core"
	TypeKubernetesPlugins = "kubernetes"
)

var (
	once       sync.Once
	pluginURLs = []string{
		// The two url will be Deprecate, but some cluster are still in use.
		"/apis/plugins.gems.cloudminds.com/v1alpha1/namespaces/gemcloud-system/installers/plugin-installer",
		"/apis/plugins.gems.cloudminds.com/v1alpha1/namespaces/kubegems-installer/installers/kubegems-plugins",

		"/apis/plugins.kubegems.io/v1beta1/namespaces/kubegems-installer/installers/kubegems-plugins",
	}
	realPluginURL string // real plugin resource position
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
	Skip         bool        `json:"skip"`
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

func GetPlugins(clus cluster.Interface) (*Plugins, error) {
	ctx := context.TODO()
	once.Do(func() {
		for _, pluginURL := range pluginURLs {
			_, err := clus.Discovery().RESTClient().Get().AbsPath(pluginURL).Do(ctx).Raw()
			if err == nil {
				realPluginURL = pluginURL
				log.Infof("use plugin url: %s", pluginURL)
				return
			}
			log.Errorf("plugins url %s error: %v", pluginURL, err)
		}
		log.Fatalf("all plugin urls failed, please check installer position")
	})

	obj, err := clus.Discovery().RESTClient().Get().AbsPath(realPluginURL).Do(ctx).Raw()
	if err != nil {
		log.Errorf("error getting plugins: %v", err)
		return nil, err
	}

	gemsplugins := &Plugins{}
	if err := json.Unmarshal(obj, gemsplugins); err != nil {
		log.Errorf("error unmarshalling plugins: %v", err)
	}
	return gemsplugins, nil
}

func UpdatePlugins(clus cluster.Interface, plugins *Plugins) error {
	ctx := context.TODO()
	obj, err := json.Marshal(plugins)
	if err != nil {
		return err
	}
	_, err = clus.Discovery().RESTClient().Put().AbsPath(realPluginURL).Body(obj).DoRaw(ctx)
	if err != nil {
		log.Errorf("error update plugins: %v", err)
		return err
	}
	return nil
}

func IsPluginHelthy(clus cluster.Interface, plugin *Plugin) bool {
	if !plugin.Enabled {
		return false
	}
	if len(plugin.Deployment)+len(plugin.Statefulset)+len(plugin.Daemonset) == 0 {
		return false
	}

	ctx := context.TODO()
	for _, v := range plugin.Deployment {
		obj := appsv1.Deployment{}
		err := clus.GetClient().Get(ctx, types.NamespacedName{
			Namespace: plugin.Namespace,
			Name:      v,
		}, &obj)
		if err != nil || obj.Spec.Replicas == nil || obj.Status.ReadyReplicas != *obj.Spec.Replicas {
			return false
		}
	}
	for _, v := range plugin.Statefulset {
		obj := appsv1.StatefulSet{}
		err := clus.GetClient().Get(ctx, types.NamespacedName{
			Namespace: plugin.Namespace,
			Name:      v,
		}, &obj)
		if err != nil || obj.Spec.Replicas == nil || obj.Status.ReadyReplicas != *obj.Spec.Replicas {
			return false
		}
	}
	for _, v := range plugin.Daemonset {
		obj := appsv1.DaemonSet{}
		err := clus.GetClient().Get(ctx, types.NamespacedName{
			Namespace: plugin.Namespace,
			Name:      v,
		}, &obj)
		if err != nil || obj.Status.NumberReady != obj.Status.DesiredNumberScheduled {
			return false
		}
	}
	return true
}
