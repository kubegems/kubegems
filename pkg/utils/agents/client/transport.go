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
	"net/http"
	"net/http/httputil"
	"net/url"

	"kubegems.io/kubegems/pkg/utils/httpsigs"
	"kubegems.io/library/net/httpproxy"
)

func (auth *Auth) IsEmpty() bool {
	return auth.Token == "" && auth.Username == "" && auth.Password == ""
}

func NewHTTPSigner(basepath string) func(req *http.Request) error {
	signer := httpsigs.GetSigner()
	return func(req *http.Request) error {
		signer.Sign(req, basepath)
		return nil
	}
}

func NewTokenAuth(token string) func(req *http.Request) error {
	return func(req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

func NewBasicAuth(username, password string) func(req *http.Request) error {
	return func(req *http.Request) error {
		req.SetBasicAuth(username, password)
		return nil
	}
}

// NewReverseProxy return a reverse proxy that proxy requests to the agent.
func NewReverseProxy(addr *url.URL, tp http.RoundTripper) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Transport: tp,
		Rewrite:   func(pr *httputil.ProxyRequest) { pr.SetURL(addr) },
	}
}

// NewProxyTransport return a transport that handle requests like it happens in the agent pod.
func NewProxyTransport(server *url.URL, tp http.RoundTripper) http.RoundTripper {
	serveraddr := *server
	serveraddr.Path += "/internal/proxy"
	return &httpproxy.Client{
		Server:     &serveraddr,
		HttpClient: &http.Client{Transport: tp},
	}
}
