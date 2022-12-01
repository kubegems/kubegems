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

package common

import (
	"context"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ListOptions struct {
	Page     int
	Size     int
	Search   string // name regexp
	Selector labels.Selector
}

type EdgeStore[T any] interface {
	List(ctx context.Context, options ListOptions) (int, []T, error)
	Get(ctx context.Context, name string) (T, error)
	Update(ctx context.Context, name string, fun func(cluster T) error) (T, error)
	Delete(ctx context.Context, name string) (T, error)
}

type EdgeClusterK8sStore[T client.Object] struct {
	cli     client.Client
	example T
	ns      string
}

func (s EdgeClusterK8sStore[T]) List(ctx context.Context, options ListOptions) (int, []T, error) {
	list := &unstructured.UnstructuredList{}
	list.GetObjectKind().SetGroupVersionKind(s.example.GetObjectKind().GroupVersionKind())

	listopts := []client.ListOption{
		client.InNamespace(s.ns),
	}
	if options.Selector != nil {
		listopts = append(listopts, client.MatchingLabelsSelector{Selector: options.Selector})
	}
	if err := s.cli.List(ctx, list, listopts...); err != nil {
		return 0, nil, err
	}
	if options.Page == 0 && options.Size == 0 {
		return len(list.Items), toList[T](list.Items), nil
	} else {
		searchname := func(item unstructured.Unstructured) bool {
			return options.Search == "" || strings.Contains(item.GetName(), options.Search)
		}
		paged := response.NewTypedPage(list.Items, options.Page, options.Size, searchname, nil)
		return int(paged.Total), toList[T](paged.List), nil
	}
}

func toList[T any](list []unstructured.Unstructured) []T {
	ret := make([]T, len(list))
	for i, item := range list {
		runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &ret[i])
	}
	return ret
}

func (s EdgeClusterK8sStore[T]) Get(ctx context.Context, name string) (T, error) {
	ret := s.new(name)
	if err := s.cli.Get(ctx, client.ObjectKey{Name: name, Namespace: s.ns}, ret); err != nil {
		return ret, err
	}
	return ret, nil
}

func (s EdgeClusterK8sStore[T]) Update(ctx context.Context, name string, fun func(cluster T) error) (T, error) {
	obj := s.new(name)
	if err := CreateOrUpdateWithStatus(ctx, s.cli, obj, fun); err != nil {
		return obj, err
	}
	return obj, nil
}

func (s EdgeClusterK8sStore[T]) new(name string) T {
	// nolint: forcetypeassert
	obj := s.example.DeepCopyObject().(T)
	obj.SetName(name)
	obj.SetNamespace(s.ns)
	return obj
}

// nolint: nestif,funlen,gocognit,forcetypeassert
func CreateOrUpdateWithStatus[T client.Object](ctx context.Context, cli client.Client, obj T, fun func(obj T) error) error {
	if err := cli.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// init change
		if err := fun(obj); err != nil {
			return err
		}
		initUnstructed, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.DeepCopyObject())
		if err != nil {
			return err
		}
		// Attempt to extract the status from the resource for easier comparison later
		initStatus, hasInitStatus, err := unstructured.NestedFieldCopy(initUnstructed, "status")
		if err != nil {
			return err
		}
		// create
		if err := cli.Create(ctx, obj); err != nil {
			return err
		}
		if !hasInitStatus {
			return nil
		}
		statusPatch := client.MergeFrom(obj.DeepCopyObject().(client.Object))
		after, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.DeepCopyObject())
		if err != nil {
			return err
		}
		// set status to current object
		if err = unstructured.SetNestedField(after, initStatus, "status"); err != nil {
			return err
		}
		// If Status was replaced by Patch before, restore patched structure to the obj
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(after, obj); err != nil {
			return err
		}
		// patch status
		return cli.Status().Patch(ctx, obj, statusPatch)
	}

	objPatch := client.MergeFrom(obj.DeepCopyObject().(client.Object))
	statusPatch := client.MergeFrom(obj.DeepCopyObject().(client.Object))

	before, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.DeepCopyObject())
	if err != nil {
		return err
	}
	beforeStatus, hasBeforeStatus, err := unstructured.NestedFieldCopy(before, "status")
	if err != nil {
		return err
	}
	if hasBeforeStatus {
		unstructured.RemoveNestedField(before, "status")
	}
	// update
	if err := fun(obj); err != nil {
		return err
	}
	after, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	afterStatus, hasAfterStatus, err := unstructured.NestedFieldCopy(after, "status")
	if err != nil {
		return err
	}
	if hasAfterStatus {
		unstructured.RemoveNestedField(after, "status")
	}
	if !reflect.DeepEqual(before, after) {
		// Only issue a Patch if the before and after resources (minus status) differ
		if err := cli.Patch(ctx, obj, objPatch); err != nil {
			return err
		}
	}
	if (hasBeforeStatus || hasAfterStatus) && !reflect.DeepEqual(beforeStatus, afterStatus) {
		// set previous status to current obj
		objectAfterPatch, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}
		if err = unstructured.SetNestedField(objectAfterPatch, afterStatus, "status"); err != nil {
			return err
		}
		// If Status was replaced by Patch before, restore patched structure to the obj
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(objectAfterPatch, obj); err != nil {
			return err
		}
		return cli.Status().Patch(ctx, obj, statusPatch)
	}
	return nil
}

func (s EdgeClusterK8sStore[T]) Delete(ctx context.Context, name string) (T, error) {
	remove := s.new(name)
	if err := s.cli.Delete(ctx, remove); err != nil {
		return remove, err
	}
	return remove, nil
}
