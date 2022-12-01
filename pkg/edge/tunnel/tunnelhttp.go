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

package tunnel

import (
	"net/http"
	"time"
)

// nolint: gomnd
// same with http.DefaultTransport
// use http2 rr to reuse http(tcp) connection
func (s *TunnelServer) TransportOnTunnel(dest string) http.RoundTripper {
	val, ok := s.statefultransports.Load(dest)
	if ok {
		rr, ok := val.(http.RoundTripper)
		if ok {
			return rr
		}
	}
	defaultTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           s.DialerOn(dest).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	s.statefultransports.Store(dest, defaultTransport)
	return defaultTransport
}
