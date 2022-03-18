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

func (s *Set[T]) Append(vals ...T) {
	for _, val := range vals {
		if _, ok := s.maps[val]; ok {
			continue
		}
		s.elems = append(s.elems, val)
		s.maps[val] = struct{}{}
	}
}

func (s *Set[T]) Slice() []T {
	sort.Slice(s.elems, func(i, j int) bool {
		return s.elems[i] < s.elems[j]
	})
	return s.elems
}

func (s *Set[t]) Len() int {
	return len(s.elems)
}
