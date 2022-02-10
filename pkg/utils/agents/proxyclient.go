package agents

import (
	"context"
	"net/http"
	"net/http/httputil"
	"path"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type ProxyClient struct {
	HTTPProxy       *httputil.ReverseProxy
	WebsockerDialer *websocket.Dialer
	cli             *Client
}

func NewProxyClientFrom(client *Client) *ProxyClient {
	name := client.Name
	targetURL := client.BaseAddr
	p := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Path = getRealPath(name, targetURL.Path, req)
			req.URL.Host = targetURL.Host
			req.URL.Scheme = targetURL.Scheme
		},
		Transport: client.transport.Clone(),
	}

	webdial := &websocket.Dialer{
		Proxy:            client.transport.Proxy,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig:  client.transport.TLSClientConfig,
	}

	return &ProxyClient{
		cli:             client,
		HTTPProxy:       p,
		WebsockerDialer: webdial,
	}
}

func (c *ProxyClient) ProxyHTTP(w http.ResponseWriter, r *http.Request) {
	c.HTTPProxy.ServeHTTP(w, r)
}

func (c *ProxyClient) DialWebsocket(ctx context.Context, path string) (*websocket.Conn, *http.Response, error) {
	return c.WebsockerDialer.DialContext(ctx, path, nil)
}

func getRealPath(name, targetPath string, req *http.Request) (realpath string) {
	prefix := path.Join("/v1/proxy/cluster", name)
	trimed := strings.TrimPrefix(req.URL.Path, prefix)
	if strings.HasPrefix(trimed, "/custom") {
		realpath = targetPath + trimed
	} else {
		realpath = targetPath + "/v1" + trimed
	}
	return
}
