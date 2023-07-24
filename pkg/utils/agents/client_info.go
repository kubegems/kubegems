// Copyright 2023 The kubegems.io Authors
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
	"net/url"

	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type APIServerInfoClient struct {
	APIServerHost *url.URL
	Discovery     discovery.DiscoveryInterface
}

func NewAPIServerInfoClientOrEmpty(cfg *rest.Config) *APIServerInfoClient {
	if cfg == nil {
		return &APIServerInfoClient{}
	}
	infocli, err := NewAPIServerInfoClient(cfg)
	if err != nil {
		return &APIServerInfoClient{}
	}
	return infocli
}

func NewAPIServerInfoClient(cfg *rest.Config) (*APIServerInfoClient, error) {
	apiserveraddr, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	discovery := memory.NewMemCacheClient(clientset.Discovery())
	return &APIServerInfoClient{APIServerHost: apiserveraddr, Discovery: discovery}, nil
}

func (c *APIServerInfoClient) APIServerAddr() string {
	if c.APIServerHost == nil {
		return ""
	}
	return c.APIServerHost.String()
}

func (c *APIServerInfoClient) APIServerVersion() string {
	if c.Discovery == nil {
		return ""
	}
	version, err := c.Discovery.ServerVersion()
	if err != nil {
		return ""
	}
	return version.String()
}
