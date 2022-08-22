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
	maps map[T]struct{}
}

func NewSet[T Ordered]() *Set[T] {
	return &Set[T]{
		maps: map[T]struct{}{},
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
		s.maps[val] = struct{}{}
	}
	return s
}

func (s *Set[T]) Slice() (r []T) {
	for k := range s.maps {
		r = append(r, k)
	}
	sort.Slice(r, func(i, j int) bool {
		return r[i] < r[j]
	})
	return
}

func (s *Set[T]) Remove(elems ...T) {
	for _, elem := range elems {
		delete(s.maps, elem)
	}
}

func (s *Set[T]) Len() int {
	return len(s.maps)
}
