// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agents

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes"
	"kubegems.io/kubegems/pkg/utils/kube"
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
	*ExtendClient
	websocket     *websocket.Dialer
	apiserverAddr *url.URL
	kubernetes    kubernetes.Interface
	discovery     discovery.DiscoveryInterface
}

type ClientMeta struct {
	Name      string
	BaseAddr  *url.URL
	TLSConfig *tls.Config
	Proxy     func(req *http.Request) (*url.URL, error)

	ServerInfo    serverInfo
	APIServerAddr *url.URL
}

func (c *DelegateClient) Extend() *ExtendClient {
	return c.ExtendClient
}

func (c *DelegateClient) Name() string {
	return c.ExtendClient.Name
}

func (c *DelegateClient) BaseAddr() url.URL {
	return *c.TypedClient.BaseAddr
}

func (c *DelegateClient) APIServerAddr() url.URL {
	return *c.apiserverAddr
}

func (c *DelegateClient) APIServerVersion() string {
	version, err := c.discovery.ServerVersion()
	if err != nil {
		return ""
	}
	return version.String()
}

func newClient(meta ClientMeta, kubernetes kubernetes.Interface, tracer trace.Tracer) Client {
	return &DelegateClient{
		ExtendClient: &ExtendClient{
			Name: meta.Name,
			TypedClient: &TypedClient{
				BaseAddr: meta.BaseAddr,
				HTTPClient: &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: meta.TLSConfig,
						Proxy:           meta.Proxy,
					},
				},
				RuntimeScheme: kube.GetScheme(),
				tracer:        tracer,
			},
		},
		websocket: &websocket.Dialer{
			Proxy:            meta.Proxy,
			HandshakeTimeout: 45 * time.Second,
			TLSClientConfig:  meta.TLSConfig,
		},
		apiserverAddr: meta.APIServerAddr,
		kubernetes:    kubernetes,
		discovery:     memory.NewMemCacheClient(kubernetes.Discovery()),
	}
}
