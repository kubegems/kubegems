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

package server

import (
	"context"
	"math"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"kubegems.io/kubegems/pkg/apis/edge/common"
	"kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/library/rest/response"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ListOptions struct {
	Page        int
	Size        int
	Search      string // name regexp
	Selector    labels.Selector
	Manufacture labels.Selector
}

type EdgeClusterStore interface {
	List(ctx context.Context, options ListOptions) (int, []v1beta1.EdgeCluster, error)
	Get(ctx context.Context, name string) (*v1beta1.EdgeCluster, error)
	Update(ctx context.Context, name string, fun func(cluster *v1beta1.EdgeCluster) error) (*v1beta1.EdgeCluster, error)
	Delete(ctx context.Context, name string) (*v1beta1.EdgeCluster, error)
}

type EdgeClusterK8sStore struct {
	cli client.Client
	ns  string
}

func (s EdgeClusterK8sStore) List(ctx context.Context, options ListOptions) (int, []v1beta1.EdgeCluster, error) {
	list := &v1beta1.EdgeClusterList{}
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
		options.Size = math.MaxInt // like do not page
		options.Page = 1
	}
	searchfunc := func(item v1beta1.EdgeCluster) bool {
		// match search name or device id
		return ((options.Search == "" || strings.Contains(item.GetName(), options.Search)) ||
			(options.Search == "" ||
				item.Status.Manufacture == nil ||
				item.Status.Manufacture[common.AnnotationKeyDeviceID] == "" ||
				strings.Contains(item.Status.Manufacture[common.AnnotationKeyDeviceID], options.Search))) &&
			//  match label selector
			(options.Manufacture == nil || options.Manufacture.Matches(labels.Set(item.Status.Manufacture)))
	}
	sortfunc := func(a, b v1beta1.EdgeCluster) int {
		return b.CreationTimestamp.Compare(a.CreationTimestamp.Time)
	}
	paged := response.PageFrom(list.Items, options.Page, options.Size, searchfunc, sortfunc)
	return int(paged.Total), paged.List, nil
}

func (s EdgeClusterK8sStore) Get(ctx context.Context, name string) (*v1beta1.EdgeCluster, error) {
	ret := &v1beta1.EdgeCluster{}
	if err := s.cli.Get(ctx, client.ObjectKey{Name: name, Namespace: s.ns}, ret); err != nil {
		return ret, err
	}
	return ret, nil
}

func (s EdgeClusterK8sStore) Update(ctx context.Context, name string, fun func(cluster *v1beta1.EdgeCluster) error) (*v1beta1.EdgeCluster, error) {
	obj := &v1beta1.EdgeCluster{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: s.ns,
		},
	}
	if err := CreateOrUpdateWithStatus(ctx, s.cli, obj, fun); err != nil {
		return obj, err
	}
	return obj, nil
}

func (s EdgeClusterK8sStore) Delete(ctx context.Context, name string) (*v1beta1.EdgeCluster, error) {
	remove := &v1beta1.EdgeCluster{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: s.ns,
		},
	}
	if err := s.cli.Delete(ctx, remove); err != nil {
		return remove, err
	}
	return remove, nil
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
