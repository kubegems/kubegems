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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/pluginmanager"
	"kubegems.io/library/rest/response"
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
		BaseClient: BaseClient{
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
	BaseClient
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

type BaseClient struct {
	Server              string
	CompleteRequestFunc func(req *http.Request)
	ErrorDecodeFunc     func(resp *http.Response) error
	DataDecoderWrapper  func(data any) any
}

func (c *BaseClient) Request(ctx context.Context, method string, path string, queries map[string]string, data interface{}, into interface{}) error {
	var body io.Reader

	switch typed := data.(type) {
	case []byte:
		body = bytes.NewReader(typed)
	case nil:
	default:
		bts, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		body = bytes.NewReader(bts)
	}
	if len(queries) != 0 {
		vals := url.Values{}
		for k, v := range queries {
			vals.Set(k, v)
		}
		path += "?" + vals.Encode()
	}
	req, err := http.NewRequest(method, c.Server+path, body)
	if err != nil {
		return err
	}
	if c.CompleteRequestFunc != nil {
		c.CompleteRequestFunc(req)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// not 200~
	// nolint: nestif
	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusIMUsed {
		if c.ErrorDecodeFunc != nil {
			return c.ErrorDecodeFunc(resp)
		}
		// nolint: gomnd
		errmsg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return errors.New(string(errmsg))
	}
	if into == nil {
		return nil
	}

	if c.DataDecoderWrapper != nil {
		into = c.DataDecoderWrapper(into)
	}
	return json.NewDecoder(resp.Body).Decode(into)
}
