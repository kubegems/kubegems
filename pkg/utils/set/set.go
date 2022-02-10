package set

import "sort"

// Set 是一个*有序的*，非空的，不重复的 string 元素的集合。
type Set struct {
	elems []string
	maps  map[string]struct{}
}

func NewSet() *Set {
	return &Set{
		elems: []string{},
		maps:  map[string]struct{}{},
	}
}

func (s *Set) Append(vals ...string) {
	for _, val := range vals {
		if val == "" {
			continue
		}
		if _, ok := s.maps[val]; ok {
			return
		}
		s.elems = append(s.elems, val)
		s.maps[val] = struct{}{}
	}
}

func (s *Set) Slice() []string {
	sort.Strings(s.elems)
	return s.elems
}

func (s *Set) Len() int {
	return len(s.elems)
}
