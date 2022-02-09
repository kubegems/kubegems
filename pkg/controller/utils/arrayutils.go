package utils

import (
	"sort"

	"github.com/google/go-cmp/cmp"
)

func StringIn(obj string, target []string) bool {
	for idx := range target {
		if target[idx] == obj {
			return true
		}
	}
	return false
}

func DelStringFromArray(obj string, target []string) []string {
	index := -1
	for idx := range target {
		if target[idx] == obj {
			index = idx
			break
		}
	}
	if index == -1 {
		return target
	}
	return append(target[:index], target[index+1:]...)
}

func StringArrayEqual(s1, s2 []string) bool {
	trans := cmp.Transformer("Sort", func(in []string) []string {
		out := append([]string(nil), in...)
		sort.Strings(out)
		return out
	})

	x := struct{ Strings []string }{s1}
	y := struct{ Strings []string }{s2}
	return cmp.Equal(x, y, trans)
}

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
