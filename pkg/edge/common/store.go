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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ListOptions struct {
	Page     int
	Size     int
	Search   string // name regexp
	Selector labels.Selector
}

type EdgeClusterStore interface {
	List(ctx context.Context, options ListOptions) (int, []v1beta1.EdgeCluster, error)
	Get(ctx context.Context, name string) (*v1beta1.EdgeCluster, error)
	Update(ctx context.Context, name string, fun func(cluster *v1beta1.EdgeCluster) error) (*v1beta1.EdgeCluster, error)
	Delete(ctx context.Context, name string) (*v1beta1.EdgeCluster, error)
}

func NewLocalK8sStore(ns string) (*EdgeClusterK8sStore, error) {
	if ns == "" {
		ns = "kubegems-edge"
	}
	cli, err := kube.NewLocalClient()
	if err != nil {
		return nil, err
	}
	return &EdgeClusterK8sStore{C: cli, NS: ns}, nil
}

type EdgeClusterK8sStore struct {
	C  client.Client
	NS string
}

func (s *EdgeClusterK8sStore) List(ctx context.Context, options ListOptions) (int, []v1beta1.EdgeCluster, error) {
	list := &v1beta1.EdgeClusterList{}
	if err := s.C.List(ctx, list,
		client.InNamespace(s.NS),
		client.MatchingLabelsSelector{Selector: options.Selector},
	); err != nil {
		return 0, nil, err
	}
	if options.Page == 0 && options.Size == 0 {
		return len(list.Items), list.Items, nil
	} else {
		searchname := func(item v1beta1.EdgeCluster) bool {
			return options.Search == "" || strings.Contains(item.Name, options.Search)
		}
		paged := response.NewTypedPage(list.Items, options.Page, options.Size, searchname, nil)
		return len(list.Items), paged.List, nil
	}
}

func (s *EdgeClusterK8sStore) Get(ctx context.Context, name string) (*v1beta1.EdgeCluster, error) {
	ret := &v1beta1.EdgeCluster{}
	if err := s.C.Get(ctx, client.ObjectKey{Name: name, Namespace: s.NS}, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *EdgeClusterK8sStore) Update(ctx context.Context, name string, fun func(cluster *v1beta1.EdgeCluster) error) (*v1beta1.EdgeCluster, error) {
	obj := &v1beta1.EdgeCluster{ObjectMeta: v1.ObjectMeta{Name: name, Namespace: s.NS}}
	// nolint: nestif
	if err := s.C.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		if err := fun(obj); err != nil {
			return nil, err
		}
		initstatus := *obj.Status.DeepCopy()
		// create spec
		if err := s.C.Create(ctx, obj); err != nil {
			return nil, err
		}
		if reflect.DeepEqual(obj.Status, initstatus) {
			return obj, nil
		}
		mergepactch := client.MergeFrom(obj.DeepCopy())
		obj.Status = initstatus
		// update status
		if err := s.C.Status().Patch(ctx, obj, mergepactch); err != nil {
			return nil, err
		}
		return obj, nil
	}
	mergespec := client.MergeFrom(obj.DeepCopy())
	if err := fun(obj); err != nil {
		return nil, err
	}
	// update spec
	changedStatus := *obj.Status.DeepCopy()
	if err := s.C.Patch(ctx, obj, mergespec); err != nil {
		return nil, err
	}
	if reflect.DeepEqual(obj.Status, changedStatus) {
		return obj, nil
	}
	// update status
	mergestatus := client.MergeFrom(obj.DeepCopy())
	obj.Status = changedStatus
	if err := s.C.Status().Patch(ctx, obj, mergestatus); err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *EdgeClusterK8sStore) Delete(ctx context.Context, name string) (*v1beta1.EdgeCluster, error) {
	remove := &v1beta1.EdgeCluster{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: s.NS},
	}
	if err := s.C.Delete(ctx, remove); err != nil {
		return nil, err
	}
	return remove, nil
}
