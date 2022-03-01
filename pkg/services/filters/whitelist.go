package filters

import (
	"strings"

	"github.com/emicklei/go-restful/v3"
)

func isOpen(req *restful.Request) bool {

	if strings.HasPrefix(req.Request.URL.Path, "/docs") || strings.HasPrefix(req.Request.URL.Path, "/v2/login") {
		return true
	}
	return false
}
