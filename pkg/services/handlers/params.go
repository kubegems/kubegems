package handlers

import "github.com/emicklei/go-restful/v3"

var (
	QueryPageNum  = restful.QueryParameter("page", "page number")
	QueryPageSize = restful.QueryParameter("size", "page size")
	QuerySearch   = restful.QueryParameter("search", "search condition")
	QueryOrder    = restful.QueryParameter("order", "order")
)

func ListCommonQuery(rb *restful.RouteBuilder) *restful.RouteBuilder {
	return rb.Param(QueryPageNum).Param(QueryPageSize).Param(QuerySearch).Param(QueryOrder)
}
