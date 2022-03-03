package filters

import (
	"strings"

	"github.com/emicklei/go-restful/v3"
)

var WhitePrefixList = []string{
	"/docs",
	"/v2/login",
}

func IsWhiteList(req *restful.Request) bool {
	for _, prefix := range WhitePrefixList {
		if strings.HasPrefix(req.Request.URL.Path, prefix) {
			return true
		}
	}
	return false
}
