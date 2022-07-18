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

package helm

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"helm.sh/helm/v3/pkg/chart/loader"
	helm_repo "helm.sh/helm/v3/pkg/repo"
	"kubegems.io/kubegems/pkg/apis/gems"
)

type Options struct {
	Addr string `json:"addr,omitempty" description:"chart repository url"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Addr: fmt.Sprintf("http://kubegems-chartmuseum.%s:8080", gems.NamespaceSystem),
	}
}

func MustNewChartMuseumClient(cfg *RepositoryConfig) *ChartmuseumClient {
	cli, err := NewChartMuseumClient(cfg)
	if err != nil {
		panic(err)
	}
	return cli
}

func NewChartMuseumClient(cfg *RepositoryConfig) (*ChartmuseumClient, error) {
	cli := &CommonClient{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// todo: cert key parse
					InsecureSkipVerify: cfg.SkipTLSVerify,
				},
			},
		},
		Server: cfg.URL,
	}
	if cfg.Username != "" && cfg.Password != "" {
		cli.Auth = BasicAuth(cfg.Username, cfg.Password)
	}
	return &ChartmuseumClient{
		CommonClient: cli,
		cfg:          cfg,
	}, nil
}

type ChartmuseumClient struct {
	cfg *RepositoryConfig
	*CommonClient
}

func (c *ChartmuseumClient) Health(ctx context.Context) error {
	_, err := c.doRequestWithResponse(ctx, http.MethodGet, "/health", nil, nil)
	return err
}

func (c *ChartmuseumClient) UploadChart(ctx context.Context, repo string, chartContent io.Reader) error {
	path := fmt.Sprintf("/api/%s/charts?force=true", repo)
	_, err := c.doRequestWithResponse(ctx, http.MethodPost, path, chartContent, nil)
	if err != nil {
		return fmt.Errorf("upload chart error: %w", err)
	}
	return nil
}

func (c *ChartmuseumClient) GetIndex(ctx context.Context, repo string) (*helm_repo.IndexFile, error) {
	index := &helm_repo.IndexFile{}
	_, err := c.doRequestWithResponse(ctx, http.MethodGet, fmt.Sprintf("/api/%s/index.yaml", repo), nil, index)
	return index, err
}

func (c *ChartmuseumClient) GetChartFile(ctx context.Context, repo, filename string) ([]byte, error) {
	into := bytes.Buffer{}
	_, err := c.doRequestWithResponse(ctx, http.MethodGet, fmt.Sprintf("/%s/%s", repo, filename), nil, &into)
	return into.Bytes(), err
}

func (c *ChartmuseumClient) ListAllChartVersions(ctx context.Context, repo string) (map[string]helm_repo.ChartVersions, error) {
	index := map[string]helm_repo.ChartVersions{}
	_, err := c.doRequestWithResponse(ctx, http.MethodGet, fmt.Sprintf("/api/%s/charts", repo), nil, &index)
	return index, err
}

func (c *ChartmuseumClient) ListChartVersions(ctx context.Context, repo string, name string) (*helm_repo.ChartVersions, error) {
	index := &helm_repo.ChartVersions{}
	_, err := c.doRequestWithResponse(ctx, http.MethodGet, fmt.Sprintf("/api/%s/charts/%s", repo, name), nil, index)
	return index, err
}

func (c *ChartmuseumClient) HeadChartVersion(ctx context.Context, repo, name, version string) error {
	_, err := c.doRequestWithResponse(ctx, http.MethodHead, fmt.Sprintf("/api/%s/charts/%s/%s", repo, name, version), nil, nil)
	return err
}

func (c *ChartmuseumClient) GetChartVersion(ctx context.Context, repo, name, version string) (*helm_repo.ChartVersion, error) {
	data := &helm_repo.ChartVersion{}
	_, err := c.doRequestWithResponse(ctx, http.MethodGet, fmt.Sprintf("/api/%s/charts/%s/%s", repo, name, version), nil, data)
	return data, err
}

func (c *ChartmuseumClient) GetChartBufferedFiles(ctx context.Context, repo, name, version string) ([]*loader.BufferedFile, error) {
	cv, err := c.GetChartVersion(ctx, repo, name, version)
	if err != nil {
		return nil, err
	}
	content, err := c.GetChartFile(ctx, repo, cv.URLs[0])
	if err != nil {
		return nil, err
	}
	return loader.LoadArchiveFiles(bytes.NewReader(content))
}

func (c *ChartmuseumClient) NewLegencyRepositoryClientFrom(ctx context.Context, repo string) (*LegencyRepository, error) {
	newcfg := *c.cfg
	newcfg.URL += "/" + repo
	return NewLegencyRepository(&newcfg)
}

type Auth func(req *http.Request)

func BasicAuth(username, password string) Auth {
	return func(req *http.Request) { req.SetBasicAuth(username, password) }
}

func TokenAuth(token string) Auth {
	return func(req *http.Request) { req.Header.Add("Authorization", "Bearer "+token) }
}

type CommonClient struct {
	Client *http.Client
	Server string
	Auth   Auth
}

func (c *CommonClient) doRequestWithResponse(ctx context.Context, method string, path string, data interface{}, decodeinto interface{}) (*http.Response, error) {
	var body io.Reader
	switch typed := data.(type) {
	case io.Reader:
		body = typed
	case []byte:
		body = bytes.NewReader(typed)
	case string:
		body = bytes.NewBufferString(typed)
	case nil:
	default:
		bts, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(bts)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.Server+path, body)
	if err != nil {
		return nil, err
	}

	if c.Auth != nil {
		c.Auth(req)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusIMUsed {
		bytes, _ := ioutil.ReadAll(resp.Body)
		return resp, errors.New(string(bytes))
	}

	// resp into writer
	switch into := decodeinto.(type) {
	case io.Writer:
		_, err := io.Copy(into, resp.Body)
		return resp, err
	case []byte:
		_, err := io.Copy(bytes.NewBuffer(into), resp.Body)
		return resp, err
	case *[]byte:
		_, err := io.Copy(bytes.NewBuffer(*into), resp.Body)
		return resp, err
	case nil:
	default:
		return resp, json.NewDecoder(resp.Body).Decode(decodeinto)
	}
	return resp, nil
}

type ErrorResponse struct {
	ErrorMsg string `json:"error,omitempty"`
}

func (e ErrorResponse) Error() string {
	return e.ErrorMsg
}
