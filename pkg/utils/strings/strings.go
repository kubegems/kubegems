package strings

func StrOrDef(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}

// src中是否存在了dest字符串
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
