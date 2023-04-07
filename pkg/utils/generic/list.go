package generic

func MapList[T any, F any](list []T, mapfunc func(item T) F) []F {
	result := make([]F, 0, len(list))
	for _, item := range list {
		result = append(result, mapfunc(item))
	}
	return result
}
