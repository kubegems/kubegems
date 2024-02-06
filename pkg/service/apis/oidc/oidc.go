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

package oidc

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"

	"gopkg.in/square/go-jose.v2"
	"kubegems.io/library/rest/api"
	"kubegems.io/library/rest/response"
)

const (
	DiscoveryEndpoint = "/.well-known/openid-configuration"
	JWKSPath          = "/keys"
)

// nolint: tagliatelle
type DiscoveryConfiguration struct {
	Issuer                           string   `json:"issuer,omitempty"`
	JwksURI                          string   `json:"jwks_uri,omitempty"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported,omitempty"`
}

type OIDCProvider struct {
	issuerPrefix string
	keys         *jose.JSONWebKeySet
	discovery    DiscoveryConfiguration
}

func NewProvider(ctx context.Context, options *OIDCOptions) (*OIDCProvider, error) {
	tlscert, err := tls.LoadX509KeyPair(options.CertFile, options.KeyFile)
	if err != nil {
		return nil, err
	}
	keys := &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			(&jose.JSONWebKey{Key: tlscert.PrivateKey, Algorithm: string(jose.RS256), Use: "sig"}).Public(),
		},
	}
	return &OIDCProvider{keys: keys}, nil
}

func (m *OIDCProvider) Discovery(w http.ResponseWriter, r *http.Request) {
	issuer := m.dynamicIssuer(r)
	discovery := DiscoveryConfiguration{
		Issuer:                           issuer,
		JwksURI:                          issuer + JWKSPath,
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
	}
	response.Raw(w, http.StatusOK, discovery, nil)
}

func (m *OIDCProvider) JWKS(w http.ResponseWriter, r *http.Request) {
	response.Raw(w, http.StatusOK, m.keys, nil)
}

func (m *OIDCProvider) dynamicIssuer(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host, prefix := r.Host, m.issuerPrefix

	if proxy_scheme := r.Header.Get("X-Forwarded-Proto"); proxy_scheme != "" {
		scheme = proxy_scheme
	}
	if proxy_host := r.Header.Get("X-Forwarded-Host"); proxy_host != "" {
		host = proxy_host
	}
	if proxy_uri := r.Header.Get("X-Forwarded-URI"); proxy_uri != "" {
		if uri, _ := url.ParseRequestURI(proxy_uri); uri != nil {
			prefix = strings.TrimSuffix(uri.Path, DiscoveryEndpoint)
		}
	}
	return scheme + "://" + host + prefix
}

func (m *OIDCProvider) RegisterRoute(g *api.Group) {
	g.AddRoutes(
		api.GET(JWKSPath).To(m.JWKS),
		api.GET(DiscoveryEndpoint).To(m.Discovery),
	)
}
