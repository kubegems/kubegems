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

package response

import (
	"net/http"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
)

type SortAndSearchAble interface {
	GetName() string
	GetCreationTimestamp() metav1.Time
}

type Named interface {
	GetName() string
}

type (
	TypedSortFun[T any]   func(a, b T) bool
	TypedFilterFun[T any] func(item T) bool
)

type Page struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int64       `json:"page,omitempty"`
	Size  int64       `json:"size,omitempty"`
}

type TypedPage[T any] struct {
	Total       int64
	List        []T // TODO: add lowercase fields tag here
	CurrentPage int64
	CurrentSize int64
}

func PageFromRequest[T any](req *http.Request, list []T) TypedPage[T] {
	page, size := request.Query(req, "page", 1), request.Query(req, "size", defaultPageSize)
	return NewTypedPage(list, page, size, nil, nil)
}

func NewTypedPage[T any](list []T, page, size int, pickfun func(item T) bool, sortfun func(a, b T) bool) TypedPage[T] {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultPageSize
	}

	// filter
	if pickfun != nil {
		datas := []T{}
		for _, item := range list {
			if pickfun(item) {
				datas = append(datas, item)
			}
		}
		list = datas
	}

	// sort
	if sortfun != nil {
		sort.Slice(list, func(i, j int) bool {
			return sortfun(list[i], list[j])
		})
	}

	// page
	total := len(list)
	startIdx := (page - 1) * size
	endIdx := startIdx + size
	if startIdx > total {
		startIdx = 0
		endIdx = 0
	}
	if endIdx > total {
		endIdx = total
	}
	list = list[startIdx:endIdx]
	return TypedPage[T]{
		Total:       int64(total),
		List:        list,
		CurrentPage: int64(page),
		CurrentSize: int64(size),
	}
}
