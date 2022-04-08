package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
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

func TemplatePlugins(ctx context.Context, path string, dest string, collectfun func(runtime.Object), recursive bool) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}

	var objs []runtime.Object
	// if path is a file,decode into a plugin
	if fi.IsDir() {
		name := filepath.Base(path)
		plugin := &pluginsv1beta1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Spec: pluginsv1beta1.PluginSpec{
				Enabled: true,
				Kind:    DetectPluginType(path),
			},
		}
		objs = append(objs, plugin)
	} else {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		objs, err = SplitYAMLTyped(raw)
		if err != nil {
			return err
		}
	}

	delegate := NewDelegatePluginManager(nil, &PluginOptions{PluginsDir: dest})
	return templateRecursive(ctx, delegate, objs, recursive, collectfun)
}

func templateRecursive(ctx context.Context, pm *DelegatePluginManager, objs []runtime.Object, recursive bool, collectfunc func(runtime.Object)) error {
	for _, obj := range objs {
		if plugin, ok := obj.(*pluginsv1beta1.Plugin); ok {
			uns, err := template(ctx, pm, plugin)
			if err != nil {
				return err
			}
			objs := ConvertToTyped(uns)
			if !recursive {
				for _, obj := range objs {
					collectfunc(obj)
				}
				continue
			}
			templateRecursive(ctx, pm, objs, recursive, collectfunc)
		} else {
			collectfunc(obj)
		}
	}
	return nil
}

func template(ctx context.Context, pm *DelegatePluginManager, apiplugin *pluginsv1beta1.Plugin) ([]*unstructured.Unstructured, error) {
	// template plugin
	plugin := PluginFromPlugin(apiplugin)
	plugin.DryRun = true

	pstatus := &PluginStatus{}
	if err := pm.Apply(ctx, plugin, pstatus); err != nil {
		return nil, err
	}
	return pstatus.Resources, nil
}

func SplitYAMLTyped(raw []byte) ([]runtime.Object, error) {
	const readcache = 4096
	d := kubeyaml.NewYAMLOrJSONDecoder(bytes.NewReader(raw), readcache)
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()

	var objs []runtime.Object
	for {
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return objs, fmt.Errorf("failed to unmarshal manifest: %v", err)
		}
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}

		obj, gvk, err := decoder.Decode(ext.Raw, nil, &unstructured.Unstructured{})
		if err != nil {
			// decode type error using unstructured

			return nil, err
		}
		obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind})
		objs = append(objs, obj)
	}
	return objs, nil
}

func ConvertToTyped(uns []*unstructured.Unstructured) []runtime.Object {
	typedobjs := []runtime.Object{}
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()
	for i, us := range uns {
		raw, err := yaml.Marshal(us)
		if err != nil {
			return nil
		}
		typed, gvk, err := decoder.Decode(raw, nil, nil)
		if err != nil {
			// use default
			typedobjs = append(typedobjs, uns[i])
			continue
		}
		typed.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind})
		typedobjs = append(typedobjs, typed)
	}
	return typedobjs
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
