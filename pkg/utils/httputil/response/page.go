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
	"strings"
	"time"

	"golang.org/x/exp/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
)

const DefaultPageSize = 10

type Page[T any] struct {
	Total int64 `json:"total"`
	List  []T   `json:"list"`
	Page  int64 `json:"page"`
	Size  int64 `json:"size"`
}

type Timed interface {
	GetCreationTimestamp() metav1.Time
}
type Named interface {
	GetName() string
}
type NameTimed interface {
	Timed
	Named
}

// PageObjectFromRequest used for client.Object pagination T in list
// use any of T to suit for both eg. Pod(not implement metav1.Object) and *Pod(metav1.Object)
func PageObjectFromRequest[T any](req *http.Request, list []T) Page[T] {
	getname := func(t T) string {
		if item, ok := any(t).(Named); ok {
			return item.GetName()
		}
		if item, ok := any(&t).(Named); ok {
			return item.GetName()
		}
		return ""
	}
	gettime := func(t T) time.Time {
		if item, ok := any(t).(Timed); ok {
			return item.GetCreationTimestamp().Time
		}
		if item, ok := any(&t).(Timed); ok {
			return item.GetCreationTimestamp().Time
		}
		return time.Time{}
	}
	return PageFromRequest(req, list, getname, gettime)
}

// PageFromRequest auto pagination from user request on item name or time in list
func PageFromRequest[T any](req *http.Request, list []T, namefunc func(item T) string, timefunc func(item T) time.Time) Page[T] {
	page, size := request.Query(req, "page", 1), request.Query(req, "size", DefaultPageSize)
	sort, search := request.Query(req, "sort", ""), request.Query(req, "search", "")
	return PageFrom(list, page, size, searchNameFunc(search, namefunc), sortFuncBy(sort, namefunc, timefunc))
}

func PageOnly[T any](list []T, page, size int) Page[T] {
	return PageFrom(list, page, size, nil, nil)
}

func PageFrom[T any](list []T, page, size int, pickfun func(item T) bool, sortfun func(a, b T) bool) Page[T] {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = DefaultPageSize
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
		slices.SortFunc(list, sortfun)
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
	return Page[T]{
		Total: int64(total),
		List:  list,
		Page:  int64(page),
		Size:  int64(size),
	}
}

func searchNameFunc[T any](search string, getname func(T) string) func(T) bool {
	if getname == nil || search == "" {
		return nil
	}
	return func(item T) bool {
		return strings.Contains(getname(item), search)
	}
}

func sortFuncBy[T any](by string, getname func(T) string, gettime func(T) time.Time) func(a, b T) bool {
	switch by {
	case "createTimeAsc":
		if gettime == nil {
			return nil
		}
		return func(a, b T) bool {
			tima, timb := gettime(a), gettime(b)
			if tima.Equal(timb) && getname != nil {
				return strings.Compare(getname(a), getname(b)) == -1
			}
			return tima.Before(timb)
		}
	case "createTimeDesc", "createTime", "time", "":
		if gettime == nil {
			return nil
		}
		return func(a, b T) bool {
			tima, timb := gettime(a), gettime(b)
			if tima.Equal(timb) && getname != nil {
				return strings.Compare(getname(a), getname(b)) == -1
			}
			return tima.After(timb)
		}
	case "nameDesc", "-name":
		if getname == nil {
			return nil
		}
		return func(a, b T) bool {
			return strings.Compare(getname(a), getname(b)) == 1
		}
	case "nameAsc", "name":
		if getname == nil {
			return nil
		}
		return func(a, b T) bool {
			return strings.Compare(getname(a), getname(b)) == -1
		}
	default:
		return nil
	}
}
