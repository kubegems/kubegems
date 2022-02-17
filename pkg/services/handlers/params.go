package handlers

import (
	"strconv"

	"github.com/emicklei/go-restful/v3"
)

var (
	QueryPageNum  = restful.QueryParameter("page", "page number")
	QueryPageSize = restful.QueryParameter("size", "page size")
	QuerySearch   = restful.QueryParameter("search", "search condition")
	QueryOrder    = restful.QueryParameter("order", "order")
)

func ListCommonQuery(rb *restful.RouteBuilder) *restful.RouteBuilder {
	return rb.Param(QueryPageNum).Param(QueryPageSize).Param(QuerySearch).Param(QueryOrder)
}

type IntOrString struct {
	intV  int
	strV  string
	isInt bool
}

func (i *IntOrString) IsInt() bool {
	return i.isInt
}

func (i *IntOrString) Int() int {
	return i.intV
}

func (i *IntOrString) Uint() uint {
	return uint(i.intV)
}

func (i *IntOrString) Int64() int64 {
	return int64(i.intV)
}

func (i *IntOrString) String() string {
	return i.strV
}

func (i *IntOrString) IsString() bool {
	return !i.isInt
}

func ParsePrimaryKey(v string) *IntOrString {
	r := &IntOrString{
		intV: 0,
		strV: "",
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		r.strV = v
	} else {
		r.intV = i
	}
	return r
}
