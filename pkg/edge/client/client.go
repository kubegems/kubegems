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

package client

import (
	"crypto/tls"
	"fmt"
	"net/url"

	agentclient "kubegems.io/kubegems/pkg/utils/agents/client"
	"kubegems.io/kubegems/pkg/utils/kube/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewEdgeClient creates a new EdgeClient.
func NewEdgeClient(edgeServerAddr string, uid string) (client.Client, error) {
	if uid == "" {
		return nil, fmt.Errorf("device id is empty")
	}
	u, err := url.Parse(fmt.Sprintf("%s/v1/edge-clusters/%s/proxy", edgeServerAddr, uid))
	if err != nil {
		return nil, err
	}
	clioptions := &agentclient.Config{
		Addr: u,
		TLS:  &tls.Config{InsecureSkipVerify: true},
	}
	cli := agentclient.NewTypedClient(clioptions.Addr, agentclient.ConfigAsTransport(clioptions), schema.GetScheme())
	return cli, nil
}
