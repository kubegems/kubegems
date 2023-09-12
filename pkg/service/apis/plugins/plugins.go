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

package plugins

import (
	"net/http"

	"kubegems.io/kubegems/pkg/installer/pluginmanager"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/library/rest/api"
	"kubegems.io/library/rest/response"
)

type PluginsAPI struct {
	agents *agents.ClientSet
}

type PluginsStatus struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func NewPluginsAPI(cli *agents.ClientSet) (*PluginsAPI, error) {
	return &PluginsAPI{agents: cli}, nil
}

func (p *PluginsAPI) List(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	cli, err := p.agents.ClientOfManager(ctx)
	if err != nil {
		response.OK(resp, []PluginsStatus{})
		return
	}

	installed, err := (&pluginmanager.PluginManager{Client: cli}).ListInstalled(ctx, false)
	if err != nil {
		response.Error(resp, err)
		return
	}
	ret := []PluginsStatus{}
	for name, val := range installed {
		ret = append(ret, PluginsStatus{
			Name:    name,
			Enabled: val.Enabled,
		})
	}
	response.OK(resp, ret)
}

func (p *PluginsAPI) RegisterRoute(rg *api.Group) {
	rg.Tag("plugins").AddRoutes(
		api.GET("/v1/plugins").To(p.List).Doc("List plugins").Response([]PluginsStatus{}),
	)
}
