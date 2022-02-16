package agents

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WrappedClient struct {
	Name      string
	BaseAddr  *url.URL
	Timeout   time.Duration
	transport *http.Transport

	TypedClient *TypedClient
	HttpClient  *HttpClient
}

type Client interface {
	client.WithWatch
	ClientExtend
}

type Request struct {
	Method  string
	Path    string // queries 可以放在 path 中
	Query   url.Values
	Headers http.Header
	Body    interface{}
	Into    interface{}
}

type ClientExtend interface {
	DoRequest(ctx context.Context, req Request) error
	DoRawRequest(ctx context.Context, req Request) (*http.Response, error)
	DialWebsocket(ctx context.Context, path string, headers ...http.Header) (*websocket.Conn, *http.Response, error)
	// Deprecated: remove this method
	Proxy(ctx context.Context, obj client.Object, port int, req *http.Request, writer http.ResponseWriter, rewritefunc func(r *http.Response) error) error
}

func QueryFrom(kvs map[string]string) url.Values {
	value := url.Values{}
	for k, v := range kvs {
		value.Add(k, v)
	}
	return value
}

func HeadersFrom(kvs map[string]string) http.Header {
	header := http.Header{}
	for k, v := range kvs {
		header.Add(k, v)
	}
	return header
}
