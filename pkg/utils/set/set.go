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

package set

import "sort"

// Ordered is a type constraint that matches any ordered type.
// An ordered type is one that supports the <, <=, >, and >= operators.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

// Set 是一个*有序的*，非空的，不重复的 string 元素的集合。
type Set[T Ordered] struct {
	elems []T
	maps  map[T]struct{}
}

func NewSet[T Ordered]() *Set[T] {
	return &Set[T]{
		elems: []T{},
		maps:  map[T]struct{}{},
	}
}

func (s *Set[T]) Has(value T) bool {
	_, ok := s.maps[value]
	return ok
}

func (s *Set[T]) Append(vals ...T) *Set[T] {
	for _, val := range vals {
		if _, ok := s.maps[val]; ok {
			continue
		}
		s.elems = append(s.elems, val)
		s.maps[val] = struct{}{}
	}
	return s
}

func (s *Set[T]) Slice() []T {
	sort.Slice(s.elems, func(i, j int) bool {
		return s.elems[i] < s.elems[j]
	})
	return s.elems
}

func (s *Set[T]) Len() int {
	return len(s.elems)
}
