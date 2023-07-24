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

package client

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/gorilla/websocket"
)

const (
	DefaultWebSocketHandshakeTimeout = 45 * time.Second
)

func NewWebsocketClient(options *Config) *WebsocketClient {
	return &WebsocketClient{
		BaseAddr: options.Addr,
		Dialer: &websocket.Dialer{
			TLSClientConfig:  options.TLS,
			HandshakeTimeout: DefaultWebSocketHandshakeTimeout,
			Proxy:            ConfigAuthAsProxy(options),
		},
	}
}

type WebsocketClient struct {
	BaseAddr *url.URL
	Dialer   *websocket.Dialer
}

func (c WebsocketClient) DialPath(ctx context.Context, rpath string, headers http.Header) (*websocket.Conn, *http.Response, error) {
	wsu := (&url.URL{
		Scheme: func() string {
			if c.BaseAddr.Scheme == "http" {
				return "ws"
			} else {
				return "wss"
			}
		}(),
		Host: c.BaseAddr.Host,
		Path: path.Join(c.BaseAddr.Path, rpath),
	}).String()
	return c.Dialer.DialContext(ctx, wsu, headers)
}
