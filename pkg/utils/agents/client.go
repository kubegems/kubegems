package agents

import (
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	Name      string
	BaseAddr  *url.URL
	Timeout   time.Duration
	transport *http.Transport

	TypedClient *TypedClient
	ProxyClient *ProxyClient
	HttpClient  *HttpClient
}
