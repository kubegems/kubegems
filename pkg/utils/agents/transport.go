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

	"kubegems.io/kubegems/pkg/utils/httpsigs"
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

type RoundTripperFunc func(req *http.Request) (*http.Response, error)

func (c RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return c(req)
}

// RoundTripOf
func RoundTripOf(cli Client) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return cli.DoRawRequest(req.Context(), Request{
			Method:  req.Method,
			Path:    req.URL.Path,
			Query:   req.URL.Query(),
			Headers: req.Header,
			Body:    req.Body,
		})
	})
}
