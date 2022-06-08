package handlers

import (
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/utils/slice"
	"kubegems.io/kubegems/pkg/v2/model/client"
)

const (
	page    = "page"
	size    = "size"
	order   = "order"
	search  = "search"
	preload = "preload"

	ASC  = "ASC"
	DESC = "DESC"

	gt  = "__gt"
	gte = "__gte"
	lt  = "__lt"
	lte = "__lte"
	in  = "__in"
)

func CommonOptions(req *restful.Request) []client.Option {
	opts := []client.Option{}
	pagesizeOption := PageSizeOption(req)
	if pagesizeOption != nil {
		opts = append(opts, pagesizeOption)
	}

	orderOpts := OrderOption(req)
	opts = append(opts, orderOpts...)

	searchOpt := SearchOption(req)
	if searchOpt != nil {
		opts = append(opts, searchOpt)
	}

	preloadOpt := PreloadOption(req)
	if preloadOpt != nil {
		opts = append(opts, preloadOpt)
	}
	return opts
}

func PageSizeOption(req *restful.Request) client.Option {
	query := req.Request.URL.Query()
	pagestr := query.Get(page)
	sizestr := query.Get(size)
	if pagestr == "" {
		pagestr = "1"
	}
	if sizestr == "" {
		sizestr = "10"
	}
	page, perr := strconv.ParseInt(pagestr, 10, 64)
	if perr != nil {
		page = 1
	}
	size, serr := strconv.ParseInt(sizestr, 10, 64)
	if serr != nil {
		size = 10
	}
	return client.PageSize(page, size)
}

//TODO: handler order priority, "order by id, age" vs "order by age, id"
func OrderOption(req *restful.Request) []client.Option {
	query := req.Request.URL.Query()
	orderstrs, exist := query[order]
	if !exist {
		return nil
	}
	opts := []client.Option{}
	for _, order := range orderstrs {
		var (
			orderField string
			desc       bool
		)
		if strings.HasSuffix(order, ASC) {
			orderField = strings.TrimSuffix(order, ASC)
		} else if strings.HasSuffix(order, DESC) {
			orderField = strings.TrimSuffix(order, DESC)
			desc = true
		} else {
			orderField = order
		}
		if orderField == "" {
			continue
		}
		if desc {
			opts = append(opts, client.OrderDesc(orderField))
		} else {
			opts = append(opts, client.OrderAsc(orderField))
		}
	}
	return opts
}

func PreloadOption(req *restful.Request) client.Option {
	query := req.Request.URL.Query()
	preloadstr, exist := query[preload]
	if !exist {
		return nil
	}
	return client.Preloads(preloadstr)
}

func SearchOption(req *restful.Request) client.Option {
	query := req.Request.URL.Query()
	searchStr := query.Get(search)
	if searchStr == "" {
		return nil
	}
	return client.Search(searchStr)
}

func WhereOptions(req *restful.Request, queryWhiteList []string) []client.Option {
	query := req.Request.URL.Query()
	opts := []client.Option{}
	for key, value := range query {
		switch key {
		case page, size, order, search, preload:
			continue
		default:
			if strings.HasSuffix(key, gt) {
				realKey := strings.TrimSuffix(key, gt)
				if slice.ContainStr(queryWhiteList, realKey) {
					opts = append(opts, client.Where(realKey, client.Gt, value[0]))
				}
			} else if strings.HasSuffix(key, gte) {
				realKey := strings.TrimSuffix(key, gte)
				if slice.ContainStr(queryWhiteList, realKey) {
					opts = append(opts, client.Where(realKey, client.Gte, value[0]))
				}
			} else if strings.HasSuffix(key, lt) {
				realKey := strings.TrimSuffix(key, lt)
				if slice.ContainStr(queryWhiteList, realKey) {
					opts = append(opts, client.Where(realKey, client.Lt, value[0]))
				}
			} else if strings.HasSuffix(key, lte) {
				realKey := strings.TrimSuffix(key, lte)
				if slice.ContainStr(queryWhiteList, realKey) {
					opts = append(opts, client.Where(realKey, client.Lte, value[0]))
				}
			} else if strings.HasSuffix(key, in) {
				realKey := strings.TrimSuffix(key, in)
				if slice.ContainStr(queryWhiteList, realKey) {
					opts = append(opts, client.Where(realKey, client.In, value))
				}
			} else {
				if slice.ContainStr(queryWhiteList, key) {
					opts = append(opts, client.Where(key, client.Eq, value[0]))
				}
			}
		}
	}
	return opts
}
