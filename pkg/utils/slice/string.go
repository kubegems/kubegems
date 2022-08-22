package slice

import (
	"sort"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// ContainStr src contains dest
func ContainStr(src []string, dest string) bool {
	for i := range src {
		if src[i] == dest {
			return true
		}
	}
	return false
}

func RemoveStrInReplace(src []string, dest string) []string {
	index := 0
	for i := range src {
		if src[i] != dest {
			src[index] = src[i]
			index++
		}
	}
	return src[:index]
}

func RemoveStr(src []string, dest string) []string {
	ret := []string{}
	for i := range src {
		if src[i] != dest {
			ret = append(ret, src[i])
		}
	}
	return ret
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

func SliceUniqueKey(s []string) string {
	tmp := append([]string{}, s...)
	sort.Strings(tmp)
	return strings.Join(tmp, "-")
}
