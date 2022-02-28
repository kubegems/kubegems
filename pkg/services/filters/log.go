package filters

import (
	"time"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/log"
)

func Log(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	start := time.Now()
	chain.ProcessFilter(req, resp)
	duration := time.Since(start)
	log.Info(req.Request.URL.String(), "duration", duration.String())
}
