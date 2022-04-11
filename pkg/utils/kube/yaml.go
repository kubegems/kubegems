package kube

import (
	"bytes"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

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

		obj, _, err := decoder.Decode(ext.Raw, nil, nil)
		if err != nil {
			// decode type error using unstructured
			obj = &unstructured.Unstructured{}
			if err := yaml.Unmarshal(ext.Raw, obj); err != nil {
				return nil, err
			}
		}
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
