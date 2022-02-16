package httputil

import (
	"reflect"
	"sort"
)

type Response struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type Page struct {
	List  interface{} `json:"list,omitempty"`
	Total int64       `json:"total,omitempty"`
	Page  int64       `json:"page,omitempty"`
	Size  int64       `json:"size,omitempty"`
}

type PageFilterFunc func(i int) bool

type PageSortFunc func(i, j int) bool

const defaultPageSize = 10

func NewPageData(list interface{}, page, size int, filterfn PageFilterFunc, sortfn PageSortFunc) Page {
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
		return Page{}
	}

	// filter
	if filterfn != nil {
		ret := reflect.MakeSlice(v.Type(), 0, size)
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

	return Page{
		List:  v.Interface(),
		Total: int64(total),
		Page:  int64(page),
		Size:  int64(size),
	}
}
