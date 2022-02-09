package utils

func LabelChanged(origin, newone map[string]string) bool {
	if len(origin) == 0 {
		return true
	}
	for k, v := range newone {
		tmpv, exist := origin[k]
		if !exist {
			return true
		}
		if tmpv != v {
			return true
		}
	}
	return false
}

func DeleteLabels(origin, todel map[string]string) map[string]string {
	if len(origin) == 0 {
		return origin
	}
	for k := range todel {
		delete(origin, k)
	}
	return origin
}

func GetLabels(origin map[string]string, keys []string) map[string]string {
	ret := map[string]string{}
	for _, key := range keys {
		if v, exist := origin[key]; exist {
			ret[key] = v
		}
	}
	return ret
}
