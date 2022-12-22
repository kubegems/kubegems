// Copyright 2022 *v1beta1.EdgeHubhe kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WI*v1beta1.EdgeHubHOU*v1beta1.EdgeHub WARRAN*v1beta1.EdgeHubIES OR CONDI*v1beta1.EdgeHubIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"context"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EdgeHubStore interface {
	List(ctx context.Context, options ListOptions) (int, []v1beta1.EdgeHub, error)
	Get(ctx context.Context, name string) (*v1beta1.EdgeHub, error)
	Update(ctx context.Context, name string, fun func(cluster *v1beta1.EdgeHub) error) (*v1beta1.EdgeHub, error)
	Delete(ctx context.Context, name string) (*v1beta1.EdgeHub, error)
}

type EdgeHubK8sStore struct {
	cli client.Client
	ns  string
}

func (s EdgeHubK8sStore) List(ctx context.Context, options ListOptions) (int, []v1beta1.EdgeHub, error) {
	list := &v1beta1.EdgeHubList{}
	listopts := []client.ListOption{
		client.InNamespace(s.ns),
	}
	if options.Selector != nil {
		listopts = append(listopts, client.MatchingLabelsSelector{Selector: options.Selector})
	}
	if err := s.cli.List(ctx, list, listopts...); err != nil {
		return 0, nil, err
	}

	filterfunc := func(item v1beta1.EdgeHub) bool {
		return (options.Search == "" || strings.Contains(item.GetName(), options.Search)) &&
			(options.Manufacture == nil || options.Manufacture.Matches(labels.Set(item.Status.Manufacture)))
	}

	if options.Page == 0 && options.Size == 0 {
		filtered := []v1beta1.EdgeHub{}
		for _, item := range list.Items {
			if filterfunc(item) {
				filtered = append(filtered, item)
			}
		}
		return len(filtered), filtered, nil
	} else {
		paged := response.NewTypedPage(list.Items, options.Page, options.Size, filterfunc, nil)
		return int(paged.Total), paged.List, nil
	}
}

func (s EdgeHubK8sStore) Get(ctx context.Context, name string) (*v1beta1.EdgeHub, error) {
	ret := &v1beta1.EdgeHub{}
	if err := s.cli.Get(ctx, client.ObjectKey{Name: name, Namespace: s.ns}, ret); err != nil {
		return ret, err
	}
	return ret, nil
}

func (s EdgeHubK8sStore) Update(ctx context.Context, name string, fun func(cluster *v1beta1.EdgeHub) error) (*v1beta1.EdgeHub, error) {
	obj := &v1beta1.EdgeHub{ObjectMeta: v1.ObjectMeta{Name: name, Namespace: s.ns}}
	if err := CreateOrUpdateWithStatus(ctx, s.cli, obj, fun); err != nil {
		return obj, err
	}
	return obj, nil
}

func (s EdgeHubK8sStore) Delete(ctx context.Context, name string) (*v1beta1.EdgeHub, error) {
	remove := &v1beta1.EdgeHub{}
	if err := s.cli.Delete(ctx, remove); err != nil {
		return remove, err
	}
	return remove, nil
}
