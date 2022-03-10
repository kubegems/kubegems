package agents

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AgentModeApiServer = "apiServerProxy"
	AgentModeAHTTP     = "http"
	AgentModeHTTPS     = "https"
)

type Client interface {
	client.WithWatch
	DoRequest(ctx context.Context, req Request) error
	DoRawRequest(ctx context.Context, clientreq Request) (*http.Response, error)
	DialWebsocket(ctx context.Context, path string, headers http.Header) (*websocket.Conn, *http.Response, error)
	Extend() *ExtendClient
	Name() string
	BaseAddr() url.URL
	APIServerAddr() url.URL
	APIServerVersion() string
	// Deprecated: remove
	Proxy(ctx context.Context, obj client.Object, port int, req *http.Request, writer http.ResponseWriter, rewritefunc func(r *http.Response) error) error
}

var _ Client = &DelegateClient{}

type DelegateClient struct {
	*TypedClient
	extend *ExtendClient
}

type ClientMeta struct {
	Name      string
	BaseAddr  *url.URL
	TLSConfig *tls.Config
	Proxy     func(req *http.Request) (*url.URL, error)

	APIServerAddr    *url.URL
	APIServerVersion string
}

func (c *DelegateClient) Extend() *ExtendClient {
	return c.extend
}

func (c *DelegateClient) Name() string {
	return c.ClientMeta.Name
}

func (c *DelegateClient) BaseAddr() url.URL {
	return *c.ClientMeta.BaseAddr
}

func (c *DelegateClient) APIServerAddr() url.URL {
	return *c.ClientMeta.APIServerAddr
}

func (c *DelegateClient) APIServerVersion() string {
	return c.ClientMeta.APIServerVersion
}

func newClient(meta ClientMeta) Client {
	typed := &TypedClient{
		ClientMeta: meta,
		http: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: meta.TLSConfig,
				Proxy:           meta.Proxy,
			},
		},
		websocket: &websocket.Dialer{
			Proxy:            meta.Proxy,
			HandshakeTimeout: 45 * time.Second,
			TLSClientConfig:  meta.TLSConfig,
		},
		scheme: scheme.Scheme,
	}

	return &DelegateClient{
		TypedClient: typed,
		extend: &ExtendClient{
			TypedClient: typed,
		},
	}
}
