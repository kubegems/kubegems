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
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"
)

type ClientOptions struct {
	Addr *url.URL
	TLS  *tls.Config
	Auth Auth
}

type Auth struct {
	EnableHttpSign *bool
	Token          string
	Username       string
	Password       string
}

func TLSConfigFrom(ca, cert, key []byte) (*tls.Config, error) {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		caCertPool = x509.NewCertPool()
	}
	if len(ca) > 0 {
		_ = caCertPool.AppendCertsFromPEM(ca)
	}
	tlsconfig := &tls.Config{RootCAs: caCertPool}
	if len(cert) > 0 && len(key) > 0 {
		certificate, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
		tlsconfig.Certificates = append(tlsconfig.Certificates, certificate)
	}
	return tlsconfig, nil
}

func OptionAuthAsProxy(options *ClientOptions) func(*http.Request) (*url.URL, error) {
	intercepters := []func(*http.Request) error{}
	auth := options.Auth
	// 默认开启 HTTP 签名
	if enable := auth.EnableHttpSign; enable == nil || *enable {
		intercepters = append(intercepters, NewHTTPSigner(options.Addr.Path))
	}
	if auth.Token != "" {
		intercepters = append(intercepters, NewTokenAuth(auth.Token))
	}
	if auth.Username != "" && auth.Password != "" {
		intercepters = append(intercepters, NewBasicAuth(auth.Username, auth.Password))
	}
	return func(r *http.Request) (*url.URL, error) {
		for _, f := range intercepters {
			if err := f(r); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
}
