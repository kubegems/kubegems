package agents

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
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
	// Deprecated: remove
	Proxy(ctx context.Context, obj client.Object, port int, req *http.Request, writer http.ResponseWriter, rewritefunc func(r *http.Response) error) error
}

var _ Client = &DelegateClient{}

type DelegateClient struct {
	*TypedClient
	extend *ExtendClient
}

type ClientMeta struct {
	Name       string
	BaseAddr   *url.URL
	TlsConfig  *tls.Config
	Restconfig *rest.Config
	Signer     func(*http.Request) (*url.URL, error)
	Transport  http.RoundTripper
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

func newClient(meta ClientMeta) Client {
	typed := &TypedClient{
		ClientMeta: meta,
		http: &http.Client{
			Transport: meta.Transport,
		},
		websocket: &websocket.Dialer{
			Proxy:            meta.Signer,
			HandshakeTimeout: 45 * time.Second,
			TLSClientConfig:  meta.TlsConfig,
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
