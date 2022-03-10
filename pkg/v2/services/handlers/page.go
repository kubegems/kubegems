package handlers

import (
	"reflect"
	"sort"
	"strconv"

	restful "github.com/emicklei/go-restful/v3"
)

const defaultPageSize = 10

func NewPageDataFromContext(req *restful.Request, fulllist interface{}, pick PageFilterFunc, sortfn PageSortFunc) PageData {
	page, _ := strconv.Atoi(req.QueryParameter("page"))
	size, _ := strconv.Atoi(req.QueryParameter("size"))
	if pick == nil {
		pick = NoopPageFilterFunc
	}
	return NewPageData(fulllist, page, size, pick, sortfn)
}

type (
	PageFilterFunc func(i int) bool
	PageSortFunc   func(i, j int) bool
)

var (
	NoopPageFilterFunc = func(i int) bool { return true }
	NoopPageSortFunc   = func(i, j int) bool { return false }
)

func NewPageData(list interface{}, page, size int, filterfn PageFilterFunc, sortfn PageSortFunc) PageData {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultPageSize
	}
	// sort
	if sortfn != nil {
		sort.Slice(list, sortfn)
	}

	v := reflect.ValueOf(list)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		return PageData{}
	}

	// filter
	if filterfn != nil {
		ret := reflect.MakeSlice(v.Type(), 0, 10)
		for i := 0; i < v.Len(); i++ {
			if filterfn(i) {
				ret = reflect.Append(ret, v.Index(i))
			}
		}
		v = ret
	}

	// page
	total := v.Len()
	start := (page - 1) * size
	end := page * size
	if end > total {
		end = total
	}
	v = v.Slice(start, end)

	return PageData{
		List:        v.Interface(),
		Total:       int64(total),
		CurrentPage: page,
		CurrentSize: size,
	}

}
