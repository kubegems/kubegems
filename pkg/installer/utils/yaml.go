// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/yaml"
)

func ReadObjectsFromFile[T runtime.Object](path string) ([]T, error) {
	filecontent, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return SplitYAMLFilterd[T](bytes.NewReader(filecontent))
}

const ReadCache = 4096

func SplitYAML(data []byte) ([]*unstructured.Unstructured, error) {
	d := kubeyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), ReadCache)
	var objs []*unstructured.Unstructured
	for {
		u := &unstructured.Unstructured{}
		if err := d.Decode(u); err != nil {
			if err == io.EOF {
				break
			}
			return objs, fmt.Errorf("failed to unmarshal manifest: %v", err)
		}
		if u.Object == nil || len(u.Object) == 0 {
			continue // skip empty object
		}
		objs = append(objs, u)
	}
	return objs, nil
}

// SplitYAMLFilterd reurns objects has type of `t`
func SplitYAMLFilterd[T runtime.Object](raw io.Reader) ([]T, error) {
	d := kubeyaml.NewYAMLOrJSONDecoder(raw, ReadCache)
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()

	var objs []T
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

		obj, gvk, err := decoder.Decode(ext.Raw, nil, nil)
		if err != nil {
			// decode type error using unstructured
			obj = &unstructured.Unstructured{}
			if e := yaml.Unmarshal(ext.Raw, obj); e != nil {
				return nil, e
			}
		}
		if gvk != nil {
			obj.GetObjectKind().SetGroupVersionKind(*gvk)
		}
		if istyped, ok := obj.(T); ok {
			objs = append(objs, istyped)
		}
	}
	return objs, nil
}

func ConvertToTypedList(uns []*unstructured.Unstructured, schema *runtime.Scheme) []client.Object {
	typedobjs := make([]client.Object, 0, len(uns))
	for _, us := range uns {
		typedobjs = append(typedobjs, ConvertToTypedObject(us, schema))
	}
	return typedobjs
}

func ConvertToTypedObject(uns *unstructured.Unstructured, schema *runtime.Scheme) client.Object {
	gvk, err := apiutil.GVKForObject(uns, schema)
	if err != nil {
		return uns
	}
	obj, err := schema.New(gvk)
	if err != nil {
		return uns
	}
	typed, ok := obj.(client.Object)
	if !ok {
		return uns
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uns.UnstructuredContent(), typed); err != nil {
		return uns
	}
	typed.GetObjectKind().SetGroupVersionKind(gvk)
	return typed
}
