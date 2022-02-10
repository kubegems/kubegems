package apis

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type ServiceProxyHandler struct{}

func (sp *ServiceProxyHandler) ServiceProxy(c *gin.Context) {
	var host string
	namespace := c.Request.Header.Get("namespace")
	service := c.Request.Header.Get("service")
	port := c.Request.Header.Get("port")

	if port == "" {
		host = fmt.Sprintf("%s.%s.svc", service, namespace)
	} else {
		host = fmt.Sprintf("%s.%s.svc:%s", service, namespace, port)
	}
	targetHost := url.URL{
		Host:   host,
		Scheme: "http",
	}
	realpath := c.Param("realpath")
	if realpath == "_" {
		realpath = ""
	}
	proxyInstance := httputil.NewSingleHostReverseProxy(&targetHost)
	proxyInstance.Director = func(req *http.Request) {
		req.Host = host
		req.URL.Host = host
		req.URL.Scheme = "http"
		req.URL.Path = "/" + realpath
		req.URL.RawQuery = c.Request.URL.Query().Encode()
	}
	proxyInstance.ServeHTTP(c.Writer, c.Request)
}
