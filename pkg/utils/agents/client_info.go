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
	"context"
	"net/url"

	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"kubegems.io/kubegems/pkg/utils/agents/extend"
)

type APIServerInfoClient interface {
	APIServerAddr() string
	APIServerVersion() string
}

type EmptyAPIServerInfoClient struct{}

func (c EmptyAPIServerInfoClient) APIServerAddr() string    { return "" }
func (c EmptyAPIServerInfoClient) APIServerVersion() string { return "" }

type KubeConfigAPIServerInfoClient struct {
	APIServerHost *url.URL
	Discovery     discovery.DiscoveryInterface
}

func NewAPIServerInfoClientOrEmpty(ext *extend.ExtendClient, cfg *rest.Config) APIServerInfoClient {
	if cfg == nil {
		return AgentAPIServerInfoClient{ExtendClient: ext}
	}
	infocli, err := NewAPIServerInfoClient(cfg)
	if err != nil {
		return EmptyAPIServerInfoClient{}
	}
	return infocli
}

func NewAPIServerInfoClient(cfg *rest.Config) (*KubeConfigAPIServerInfoClient, error) {
	apiserveraddr, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	discovery := memory.NewMemCacheClient(clientset.Discovery())
	return &KubeConfigAPIServerInfoClient{APIServerHost: apiserveraddr, Discovery: discovery}, nil
}

func (c *KubeConfigAPIServerInfoClient) APIServerAddr() string {
	if c.APIServerHost == nil {
		return ""
	}
	return c.APIServerHost.String()
}

func (c *KubeConfigAPIServerInfoClient) APIServerVersion() string {
	if c.Discovery == nil {
		return ""
	}
	version, err := c.Discovery.ServerVersion()
	if err != nil {
		return ""
	}
	return version.String()
}

var _ APIServerInfoClient = AgentAPIServerInfoClient{}

type AgentAPIServerInfoClient struct {
	ExtendClient *extend.ExtendClient
}

// APIServerAddr implements APIServerInfoClient.
func (a AgentAPIServerInfoClient) APIServerAddr() string {
	// there is no apiserver address when connect cluster via agent.
	return "https://kubernets.default.svc"
}

// APIServerVersion implements APIServerInfoClient.
func (a AgentAPIServerInfoClient) APIServerVersion() string {
	info, err := a.ExtendClient.KubernetesVersion(context.Background())
	if err != nil {
		return ""
	}
	return info.String()
}
