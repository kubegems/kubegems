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

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/pluginmanager"
	"kubegems.io/kubegems/pkg/utils/httputil/clientutil"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

type ClientOptions struct {
	Addr string `json:"addr,omitempty"`
}

func NewDefaultClientOptions() *ClientOptions {
	return &ClientOptions{
		Addr: "http://kubegems-installer.kubegems-installer:8080",
	}
}

func NewPluginsClient(server string) (*PluginsClient, error) {
	return &PluginsClient{
		BaseClient: clientutil.BaseClient{
			Server: server,
			DataDecoderWrapper: func(data any) any {
				return &response.Response{Data: data}
			},
			ErrorDecodeFunc: func(resp *http.Response) error {
				wrapper := response.Response{}
				json.NewDecoder(resp.Body).Decode(&wrapper)
				return &response.StatusError{Status: resp.StatusCode, Message: wrapper.Message}
			},
		},
	}, nil
}

type PluginsClient struct {
	clientutil.BaseClient
}

func (c *PluginsClient) ListPlugins(ctx context.Context) (map[string]pluginmanager.Plugin, error) {
	ret := map[string]pluginmanager.Plugin{}
	if err := c.BaseClient.Request(ctx, http.MethodGet, "/v1/plugins", nil, nil, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *PluginsClient) CheckUpdate(ctx context.Context) (map[string]pluginmanager.Plugin, error) {
	ret := map[string]pluginmanager.Plugin{}
	if err := c.BaseClient.Request(ctx, http.MethodGet, "/v1/plugins", map[string]string{"check-update": "true"}, nil, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *PluginsClient) GetPluginVersion(ctx context.Context,
	name, version string,
	withSchema bool, withDpendeciesCheck bool,
) (*pluginmanager.PluginVersion, error) {
	ret := &pluginmanager.PluginVersion{}
	queries := map[string]string{
		"version": version,
		"schema":  strconv.FormatBool(withSchema),
		"check":   strconv.FormatBool(withDpendeciesCheck),
	}
	if err := c.BaseClient.Request(ctx, http.MethodGet, "/v1/plugins/"+name, queries, nil, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *PluginsClient) Install(ctx context.Context, name string, version string, values map[string]any) error {
	queries := map[string]string{"version": version}
	body := pluginmanager.PluginVersion{
		Values: v1beta1.Values{Object: values},
	}
	return c.BaseClient.Request(ctx, http.MethodPut, "/v1/plugins/"+name, queries, body, nil)
}

func (c *PluginsClient) UnInstall(ctx context.Context, name string) error {
	return c.BaseClient.Request(ctx, http.MethodDelete, "/v1/plugins/"+name, nil, nil, nil)
}
