package agents

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WrappedClient struct {
	Name      string
	BaseAddr  *url.URL
	Timeout   time.Duration
	transport *http.Transport

	TypedClient *TypedClient
	ProxyClient *ProxyClient
	HttpClient  *HttpClient
}

type Client interface {
	client.Client
	ClientExtend
}

type ClientExtend interface {
	DoRequest(ctx context.Context, method, path string, body io.Reader, into interface{}) error
	DoRawRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error)
}
