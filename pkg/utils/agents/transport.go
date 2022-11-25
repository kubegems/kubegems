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
	"net/url"

	"kubegems.io/kubegems/pkg/utils/httpsigs"
)

type AuthInfo struct {
	ClientCertificate []byte `json:"clientCertificate,omitempty"`
	ClientKey         []byte `json:"clientKey,omitempty"`
	Token             string `json:"token,omitempty"`
	Username          string `json:"username,omitempty"`
	Password          string `json:"password,omitempty"`
}

func (auth *AuthInfo) IsEmpty() bool {
	return len(auth.ClientCertificate) == 0 && len(auth.ClientKey) == 0 && auth.Token == "" && auth.Username == "" && auth.Password == ""
}

func (auth *AuthInfo) Proxy(req *http.Request) (*url.URL, error) {
	if auth.Token != "" {
		req.Header.Set("Authorization", "Bearer "+auth.Token)
		return nil, nil
	}
	if _, _, exist := req.BasicAuth(); !exist && auth.Username != "" {
		req.SetBasicAuth(auth.Username, auth.Password)
		return nil, nil
	}
	return nil, nil
}

func httpSigner(basepath string) func(req *http.Request) (*url.URL, error) {
	signer := httpsigs.GetSigner()
	return func(req *http.Request) (*url.URL, error) {
		signer.Sign(req, basepath)
		return nil, nil
	}
}

type ChainedProxy []func(*http.Request) (*url.URL, error)

func (pc ChainedProxy) Proxy(req *http.Request) (*url.URL, error) {
	var finalurl *url.URL
	for _, p := range pc {
		if p == nil {
			continue
		}
		url, err := p(req)
		if err != nil {
			return nil, err
		}
		if url != nil {
			finalurl = url
		}
	}
	return finalurl, nil
}
