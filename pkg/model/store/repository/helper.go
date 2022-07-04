package repository

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
)

type CommonListOptions struct {
	Page   int64  `json:"page,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Search string `json:"search,omitempty"`

	// sort string, eg: "-name,-creationtime", "name,-creationtime"
	// the '-' prefix means descending,otherwise ascending
	Sort string `json:"sort,omitempty"`
}

// nolint: gomnd
func ParseCommonListOptions(r *restful.Request) CommonListOptions {
	return CommonListOptions{
		Page:   request.Query(r.Request, "page", int64(1)),
		Size:   request.Query(r.Request, "size", int64(10)),
		Search: request.Query(r.Request, "search", ""),
		Sort:   request.Query(r.Request, "sort", ""),
	}
}
