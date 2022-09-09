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

package system

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
)

type Options struct {
	Listen   string `json:"listen,omitempty" description:"listen address"`
	Locale   string `json:"locale,omitempty" description:"default locale for site"`
	CAFile   string `json:"caFile,omitempty" description:"ca file path"`
	CertFile string `json:"certFile,omitempty" description:"cert file path"`
	KeyFile  string `json:"keyFile,omitempty" description:"key file path"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Listen:   ":8080",
		Locale:   "",
		CAFile:   "",
		CertFile: "",
		KeyFile:  "",
	}
}

func (o *Options) IsTLSConfigEnabled() bool {
	return (o.CertFile != "" && o.KeyFile != "")
}

func (o *Options) ToTLSConfig() (*tls.Config, error) {
	cafile, certfile, keyfile := o.CAFile, o.CertFile, o.KeyFile

	config := &tls.Config{
		ClientCAs: x509.NewCertPool(),
	}

	// ca
	if cafile != "" {
		capem, err := ioutil.ReadFile(cafile)
		if err != nil {
			return nil, err
		}
		config.ClientCAs.AppendCertsFromPEM(capem)
	}

	// cert
	certificate, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return nil, err
	}
	config.Certificates = append(config.Certificates, certificate)

	return config, nil
}
