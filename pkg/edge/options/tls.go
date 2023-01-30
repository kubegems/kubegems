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

package options

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

type TLS struct {
	CertFile           string `json:"certFile,omitempty"`
	KeyFile            string `json:"keyFile,omitempty"`
	CAFile             string `json:"caFile,omitempty"`
	ClientAuth         bool   `json:"clientAuth,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
}

func NewDefaultTLS() *TLS {
	return &TLS{
		CAFile:     "certs/ca.crt",
		CertFile:   "certs/tls.crt",
		KeyFile:    "certs/tls.key",
		ClientAuth: false,
	}
}

func (o TLS) ToTLSConfig() (*tls.Config, error) {
	// certs
	certificate, err := tls.LoadX509KeyPair(o.CertFile, o.KeyFile)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}
	// ca
	cas, err := x509.SystemCertPool()
	if err != nil {
		cas = x509.NewCertPool()
	}
	if o.CAFile != "" {
		capem, err := os.ReadFile(o.CAFile)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			// no nothing
		} else {
			cas.AppendCertsFromPEM(capem)
		}
	}
	config.ClientCAs = cas
	config.RootCAs = cas
	if o.ClientAuth {
		config.ClientAuth = tls.RequireAndVerifyClientCert
	}
	if o.InsecureSkipVerify {
		config.InsecureSkipVerify = true
	}
	return config, nil
}
