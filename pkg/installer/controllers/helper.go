package controllers

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/yaml"
)

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
		return kvs
	}
	return nil
}

type RESTClientGetter struct {
	config       *rest.Config
	discovery    discovery.CachedDiscoveryInterface
	mapper       meta.RESTMapper
	clientconfig clientcmd.ClientConfig
}

//
// clientcmd.RESTConfigFromKubeConfig(rawkubeconfig)
//
// NewRESTClientGetter returns a RESTClientGetter using a custom cluster config for helm config
func NewRESTClientGetter(config *rest.Config) (*RESTClientGetter, error) {
	discovery, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	restmapper, err := apiutil.NewDynamicRESTMapper(config)
	if err != nil {
		return nil, err
	}
	return &RESTClientGetter{
		config:    config,
		discovery: memory.NewMemCacheClient(discovery),
		mapper:    restmapper,
	}, nil
}

// ToRESTConfig returns restconfig
func (g RESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return g.config, nil
}

// ToDiscoveryClient returns discovery client
func (g RESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return g.discovery, nil
}

// ToRESTMapper returns a restmapper
func (g RESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	return g.mapper, nil
}

// ToRawKubeConfigLoader return kubeconfig loader as-is
func (g RESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	panic("not implemented")
}
