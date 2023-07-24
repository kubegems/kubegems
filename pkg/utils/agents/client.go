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
	"net/http"
	"net/http/httputil"
	"net/url"

	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	agentcli "kubegems.io/kubegems/pkg/utils/agents/client"
	"kubegems.io/kubegems/pkg/utils/agents/extend"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	client.WithWatch
	Name() string
	Info() *APIServerInfoClient
	Websocket() *agentcli.WebsocketClient
	Extend() *extend.ExtendClient
	RestConfig() *rest.Config
	// ReverseProxy return a new reverse proxy that proxy requests to the agent.
	// if target set, proxy to target instead of agent.
	ReverseProxy(destOverride ...*url.URL) *httputil.ReverseProxy
	// Config return the config to build a client.
	Config() *agentcli.Config
	// ProxyTransport return a transport that handle requests like it happens in the agent pod.
	ProxyTransport() http.RoundTripper
}

var _ Client = &DelegateClient{}

func NewDelegateClientClient(name string, cfg *agentcli.Config, kubecfg *rest.Config, schema *runtime.Scheme, tracer trace.Tracer) (Client, error) {
	// transport is a stateful object, h2 reuse connections cache in transport.
	// so we may share the transport in consumers.
	transport := agentcli.ConfigAsTransport(cfg)
	return &DelegateClient{
		name:           name,
		cfg:            cfg,
		transport:      transport,
		proxytransport: agentcli.NewProxyTransport(cfg.Addr, transport),
		infoclli:       NewAPIServerInfoClientOrEmpty(kubecfg),
		TypedClient:    agentcli.NewTypedClient(cfg.Addr, transport, schema),
		extcli:         extend.NewExtendClient(cfg.Addr, transport),
		wscli:          agentcli.NewWebsocketClient(cfg),
	}, nil
}

type DelegateClient struct {
	*agentcli.TypedClient
	name           string
	cfg            *agentcli.Config
	transport      http.RoundTripper
	proxytransport http.RoundTripper
	extcli         *extend.ExtendClient
	wscli          *agentcli.WebsocketClient
	infoclli       *APIServerInfoClient
	kubeconfig    *rest.Config
}

func (c *DelegateClient) Extend() *extend.ExtendClient {
	return c.extcli
}

func (c *DelegateClient) Name() string {
	return c.name
}

func (c *DelegateClient) Config() *agentcli.Config {
	cfg := *c.cfg
	return &cfg
}

func (c *DelegateClient) ProxyTransport() http.RoundTripper {
	return c.proxytransport
}

func (c *DelegateClient) Transport() http.RoundTripper {
	return c.transport
}

func (c *DelegateClient) Info() *APIServerInfoClient {
	return c.infoclli
}

func (c *DelegateClient) Websocket() *agentcli.WebsocketClient {
	return c.wscli
}


func (c *DelegateClient) RestConfig() *rest.Config {
	return c.kubeconfig
}

// ReverseProxy return a http.Handler that proxy requests to the agent.
// if target set, proxy to target instead of to agent.
func (c *DelegateClient) ReverseProxy(dests ...*url.URL) *httputil.ReverseProxy {
	if len(dests) == 0 {
		return agentcli.NewReverseProxy(c.cfg.Addr, c.transport)
	}
	// use proxy transport to proxy to another host.
	return agentcli.NewReverseProxy(dests[0], c.proxytransport)
}
