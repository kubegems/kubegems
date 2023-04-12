// Copyright 2023 The kubegems.io Authors
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
	"context"
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CacheReader is a client.Reader.
var _ client.Reader = &CacheReader{}

// CacheReader is a Reader that reads from a informer cache.
// See: https://github.com/kubernetes-sigs/controller-runtime/blob/9489eb5b6c7733d1755ed18330876c3a8e364058/pkg/cache/internal/cache_reader.go#L39
// CacheReader wraps a cache.Index to implement the client.CacheReader interface for a single type.
type CacheReader struct {
	// indexer is the underlying indexer wrapped by this cache.
	indexer cache.Indexer

	// groupVersionKind is the group-version-kind of the resource.
	groupVersionKind schema.GroupVersionKind

	// scopeName is the scope of the resource (namespaced or cluster-scoped).
	scopeName apimeta.RESTScopeName

	// disableDeepCopy indicates not to deep copy objects during get or list objects.
	// Be very careful with this, when enabled you must DeepCopy any object before mutating it,
	// otherwise you will mutate the object in the cache.
	disableDeepCopy bool
}

func NewCacheReader(indexer cache.Indexer, gvk schema.GroupVersionKind, scopeName apimeta.RESTScopeName, disableDeepCopy bool) *CacheReader {
	return &CacheReader{
		indexer:          indexer,
		groupVersionKind: gvk,
		scopeName:        scopeName,
		disableDeepCopy:  disableDeepCopy,
	}
}

// Get checks the indexer for the object and writes a copy of it if found.
func (c *CacheReader) Get(_ context.Context, key client.ObjectKey, out client.Object) error {
	if c.scopeName == apimeta.RESTScopeNameRoot {
		key.Namespace = ""
	}
	storeKey := objectKeyToStoreKey(key)

	// Lookup the object from the indexer cache
	obj, exists, err := c.indexer.GetByKey(storeKey)
	if err != nil {
		return err
	}

	// Not found, return an error
	if !exists {
		// Resource gets transformed into Kind in the error anyway, so this is fine
		return apierrors.NewNotFound(schema.GroupResource{
			Group:    c.groupVersionKind.Group,
			Resource: c.groupVersionKind.Kind,
		}, key.Name)
	}

	// Verify the result is a runtime.Object
	if _, isObj := obj.(runtime.Object); !isObj {
		// This should never happen
		return fmt.Errorf("cache contained %T, which is not an Object", obj)
	}

	if !c.disableDeepCopy {
		// nolint: forcetypeassert
		obj = obj.(runtime.Object).DeepCopyObject()
	}

	if uns, ok := out.(*unstructured.Unstructured); ok {
		content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return apierrors.NewBadRequest(fmt.Sprintf("could not convert %T to unstructured: %v", obj, err))
		}
		uns.SetUnstructuredContent(content)
		uns.SetGroupVersionKind(c.groupVersionKind)
		return nil
	}

	outVal := reflect.ValueOf(out)
	objVal := reflect.ValueOf(obj)
	if !objVal.Type().AssignableTo(outVal.Type()) {
		return fmt.Errorf("cache had type %s, but %s was asked for", objVal.Type(), outVal.Type())
	}
	reflect.Indirect(outVal).Set(reflect.Indirect(objVal))
	if !c.disableDeepCopy {
		out.GetObjectKind().SetGroupVersionKind(c.groupVersionKind)
	}
	return nil
}

// List lists items out of the indexer and writes them to out.
func (c *CacheReader) List(_ context.Context, out client.ObjectList, opts ...client.ListOption) error {
	var objs []interface{}
	var err error
	listOpts := client.ListOptions{}
	listOpts.ApplyOptions(opts)
	switch {
	case listOpts.FieldSelector != nil:
		field, val, requiresExact := RequiresExactMatch(listOpts.FieldSelector)
		if !requiresExact {
			return fmt.Errorf("non-exact field matches are not supported by the cache")
		}
		objs, err = c.indexer.ByIndex(FieldIndexName(field), KeyToNamespacedKey(listOpts.Namespace, val))
	case listOpts.Namespace != "":
		objs, err = c.indexer.ByIndex(cache.NamespaceIndex, listOpts.Namespace)
	default:
		objs = c.indexer.List()
	}
	if err != nil {
		return err
	}
	var labelSel labels.Selector
	if listOpts.LabelSelector != nil {
		labelSel = listOpts.LabelSelector
	}
	limitSet := listOpts.Limit > 0

	_, isunstructedlist := out.(*unstructured.UnstructuredList)

	runtimeObjs := make([]runtime.Object, 0, len(objs))
	for _, item := range objs {
		if limitSet && int64(len(runtimeObjs)) >= listOpts.Limit {
			break
		}
		obj, isObj := item.(runtime.Object)
		if !isObj {
			return fmt.Errorf("cache contained %T, which is not an Object", obj)
		}
		meta, err := apimeta.Accessor(obj)
		if err != nil {
			return err
		}
		if labelSel != nil {
			lbls := labels.Set(meta.GetLabels())
			if !labelSel.Matches(lbls) {
				continue
			}
		}
		var outObj runtime.Object
		if c.disableDeepCopy {
			outObj = obj
		} else {
			outObj = obj.DeepCopyObject()
			outObj.GetObjectKind().SetGroupVersionKind(c.groupVersionKind)
		}
		if isunstructedlist {
			content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(outObj)
			if err != nil {
				return apierrors.NewBadRequest(fmt.Sprintf("could not convert %T to unstructured: %v", obj, err))
			}
			uns := &unstructured.Unstructured{}
			uns.SetUnstructuredContent(content)
			uns.SetGroupVersionKind(c.groupVersionKind)
			outObj = uns
		}
		runtimeObjs = append(runtimeObjs, outObj)
	}
	return apimeta.SetList(out, runtimeObjs)
}

// objectKeyToStorageKey converts an object key to store key.
// It's akin to MetaNamespaceKeyFunc.  It's separate from
// String to allow keeping the key format easily in sync with
// MetaNamespaceKeyFunc.
func objectKeyToStoreKey(k client.ObjectKey) string {
	if k.Namespace == "" {
		return k.Name
	}
	return k.Namespace + "/" + k.Name
}

// FieldIndexName constructs the name of the index over the given field,
// for use with an indexer.
func FieldIndexName(field string) string {
	return "field:" + field
}

// noNamespaceNamespace is used as the "namespace" when we want to list across all namespaces.
const allNamespacesNamespace = "__all_namespaces"

// KeyToNamespacedKey prefixes the given index key with a namespace
// for use in field selector indexes.
func KeyToNamespacedKey(ns string, baseKey string) string {
	if ns != "" {
		return ns + "/" + baseKey
	}
	return allNamespacesNamespace + "/" + baseKey
}

// RequiresExactMatch checks if the given field selector is of the form `k=v` or `k==v`.
func RequiresExactMatch(sel fields.Selector) (field, val string, required bool) {
	reqs := sel.Requirements()
	if len(reqs) != 1 {
		return "", "", false
	}
	req := reqs[0]
	if req.Operator != selection.Equals && req.Operator != selection.DoubleEquals {
		return "", "", false
	}
	return req.Field, req.Value, true
}
