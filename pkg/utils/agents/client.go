package agents

import (
	"context"
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
	DialWebsocket(ctx context.Context, path string, headers ...http.Header) (*websocket.Conn, *http.Response, error)
	Extend() *ExtendClient
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
	Transport *http.Transport
}

func (c *DelegateClient) Extend() *ExtendClient {
	return c.extend
}

func newClient(meta ClientMeta) Client {
	transport := meta.Transport

	typed := &TypedClient{
		ClientMeta: meta,
		http: &http.Client{
			Transport: transport,
		},
		websocket: &websocket.Dialer{
			Proxy:            transport.Proxy,
			HandshakeTimeout: 45 * time.Second,
			TLSClientConfig:  transport.TLSClientConfig,
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
