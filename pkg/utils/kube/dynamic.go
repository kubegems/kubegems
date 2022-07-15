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

package kube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"kubegems.io/bundle-controller/pkg/utils"
	"kubegems.io/kubegems/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	CreateNamespace bool `json:"createNamespace,omitempty"`
}

type Option func(*Options)

func WithCreateNamespace() Option {
	return func(o *Options) {
		o.CreateNamespace = true
	}
}

func Apply[T client.Object](ctx context.Context, config *rest.Config, resources []T, options ...Option) error {
	log := logr.FromContextOrDiscard(ctx)

	opts := &Options{}
	for _, opt := range options {
		opt(opts)
	}
	cli, err := NewClient(config)
	if err != nil {
		return err
	}

	// nolint: nestif
	if opts.CreateNamespace {
		namespaces := map[string]struct{}{}
		for _, obj := range resources {
			ns := obj.GetNamespace()
			if ns == "" {
				continue
			}
			if _, ok := namespaces[ns]; ok {
				continue
			}
			if err := cli.Get(ctx, types.NamespacedName{Name: ns}, &corev1.Namespace{}); err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				namespaces[ns] = struct{}{}
			}
		}
		for name := range namespaces {
			log.Info("create namespaces", "namespace", name)
			if err := cli.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}); err != nil {
				if errors.IsAlreadyExists(err) {
					continue
				}
				return err
			}
		}
	}
	for _, obj := range resources {
		log.Info("applying...", "gvk", obj.GetObjectKind().GroupVersionKind().String(), "name", obj.GetName(), "namespace", obj.GetNamespace())
		if err := utils.ApplyResource(ctx, cli, obj, utils.ApplyOptions{ServerSideApply: true, FieldOwner: "kubegems"}); err != nil {
			return err
		}
	}
	return nil
}

func Remove[T client.Object](ctx context.Context, config *rest.Config, resources []T, options ...Option) error {
	log := logr.FromContextOrDiscard(ctx)

	opts := &Options{}
	for _, opt := range options {
		opt(opts)
	}
	cli, err := NewClient(config)
	if err != nil {
		return err
	}
	for _, obj := range resources {
		data, err := json.Marshal(obj)
		if err != nil {
			return fmt.Errorf("json marshal: %w", err)
		}
		if err := cli.Delete(ctx, obj); err != nil {
			log.Error(err, "remove object", "data", string(data))
			return err
		}
	}
	return nil
}

func NewClient(config *rest.Config) (client.Client, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	clientOptions := client.Options{Scheme: scheme.Scheme, Mapper: mapper}
	return client.New(config, clientOptions)
}

func ParseResource(raw []byte) ([]*unstructured.Unstructured, error) {
	log.Debugf(string(raw))
	resources := []*unstructured.Unstructured{}
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(raw), yamlcacheSize)
	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return resources, fmt.Errorf("decode raw: %w", err)
		}
		obj := &unstructured.Unstructured{}
		_, _, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, obj)
		if err != nil {
			return resources, fmt.Errorf("decode object: %w", err)
		}
		resources = append(resources, obj)
	}
	return resources, nil
}

const yamlcacheSize = 1024

func CreateByYamlOrJson(ctx context.Context, cfg *rest.Config, yamlOrJson []byte) error {
	resources, err := ParseResource(yamlOrJson)
	if err != nil {
		return err
	}
	return Apply(ctx, cfg, resources)
}

func DeleteByYamlOrJson(ctx context.Context, cfg *rest.Config, yamlOrJson []byte) error {
	resources, err := ParseResource(yamlOrJson)
	if err != nil {
		return err
	}
	return Remove(ctx, cfg, resources)
}
