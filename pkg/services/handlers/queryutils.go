package handlers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/model/client"
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
)

func Options(c *gin.Context) []client.Option {
	return QueryToOptions(c.Request.URL.Query())
}

func QueryToOptions(query url.Values) []client.Option {
	opts := []client.Option{}
	pagesizeOption := parsePageSize(query)
	if pagesizeOption != nil {
		opts = append(opts, pagesizeOption)
	}

	orderOpts := parseOrder(query)
	if len(orderOpts) > 0 {
		opts = append(opts, orderOpts...)
	}

	searchOpt := parseSearch(query)
	if searchOpt != nil {
		opts = append(opts, searchOpt)
	}

	preloadOpt := parsePreload(query)
	if preloadOpt != nil {
		opts = append(opts, preloadOpt)
	}

	for key, value := range query {
		switch key {
		case page, size, order, search, preload:
			continue
		default:
			var v string
			if len(value) >= 1 {
				v = value[0]
			}
			if strings.HasSuffix(v, gt) {
				opts = append(opts, client.Where(key, client.Gt, strings.TrimSuffix(v, gt)))
			} else if strings.HasSuffix(v, gte) {
				opts = append(opts, client.Where(key, client.Gte, strings.TrimSuffix(v, gte)))
			} else if strings.HasSuffix(v, lt) {
				opts = append(opts, client.Where(key, client.Lt, strings.TrimSuffix(v, lt)))
			} else if strings.HasSuffix(v, lte) {
				opts = append(opts, client.Where(key, client.Lte, strings.TrimSuffix(v, lte)))
			} else {
				opts = append(opts, client.Where(key, client.Eq, v))
			}
		}
	}
	return opts
}

func parsePageSize(query url.Values) client.Option {
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

func parseOrder(query url.Values) []client.Option {
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

func parsePreload(query url.Values) client.Option {
	preloadstr, exist := query[preload]
	if !exist {
		return nil
	}
	return client.Preloads(preloadstr)
}

func parseSearch(query url.Values) client.Option {
	searchStr := query.Get(search)
	if searchStr == "" {
		return nil
	}
	return client.Search(searchStr)
}

func GetID(c *gin.Context, key string) (uint, error) {
	kstr := c.Param(key)
	uid, err := strconv.ParseUint(kstr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id: %s", kstr)
	}
	if uid == 0 {
		return 0, fmt.Errorf("invalid id: %s", kstr)
	}
	return uint(uid), nil
}

func MustGetID(c *gin.Context, key string) uint {
	kstr := c.Param(key)
	uid, err := strconv.ParseUint(kstr, 10, 64)
	if err != nil {
		return 0
	}
	if uid == 0 {
		return 0
	}
	return uint(uid)
}
