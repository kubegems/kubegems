package helm

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/kubegems/gems/pkg/log"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

type RepositoryConfig struct {
	Name                  string `json:"name"`
	URL                   string `json:"url"`
	Username              string `json:"username"`
	Password              string `json:"password"`
	Cert                  []byte `json:"cert"`
	Key                   []byte `json:"key"`
	CA                    []byte `json:"ca"`
	InsecureSkipTLSverify bool   `json:"insecure_skip_tls_verify"`
	PassCredentialsAll    bool   `json:"pass_credentials_all"`
}

type LegencyRepository struct {
	client *http.Client
	server *url.URL
	auth   Auth
}

func NewLegencyRepository(cfg *RepositoryConfig) (*LegencyRepository, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid chart URL format: %s", cfg.URL)
	}

	repository := &LegencyRepository{
		server: u,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// todo: cert key parse
					InsecureSkipVerify: cfg.InsecureSkipTLSverify,
				},
			},
		},
	}

	if cfg.Username != "" && cfg.Password != "" {
		repository.auth = BasicAuth(cfg.Username, cfg.Password)
	}

	return repository, nil
}

// https://github.com/helm/helm/blob/29d273f985306bc508b32455d77894f3b1eb8d4d/pkg/repo/chartrepo.go#L118
func (r *LegencyRepository) GetIndex(ctx context.Context) (*repo.IndexFile, error) {
	iocloser, err := r.GetFile(ctx, "index.yaml")
	if err != nil {
		return nil, err
	}
	defer iocloser.Close()
	index, err := ioutil.ReadAll(iocloser)
	if err != nil {
		return nil, err
	}
	i := &repo.IndexFile{}
	if err := yaml.UnmarshalStrict(index, i); err != nil {
		return i, err
	}
	if i.APIVersion == "" {
		return i, repo.ErrNoAPIVersion
	}
	for name, cvs := range i.Entries {
		for idx := len(cvs) - 1; idx >= 0; idx-- {
			if cvs[idx].APIVersion == "" {
				cvs[idx].APIVersion = chart.APIVersionV1
			}
			if err := cvs[idx].Validate(); err != nil {
				log.Warnf("skipping loading invalid entry for chart %q %q from %s: %s",
					name, cvs[idx].Version, r.server.String(), err)
				cvs = append(cvs[:idx], cvs[idx+1:]...)
			}
		}
	}
	i.SortEntries()
	return i, nil
}

// GetChart 如果 version 为空，则使用最新版本
func (r *LegencyRepository) GetFile(ctx context.Context, url string) (io.ReadCloser, error) {
	// 应对相对路径的 url
	if !strings.HasPrefix(url, "http") {
		url = r.server.String() + "/" + url
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if r.auth != nil {
		r.auth(req)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusBadRequest {
		errbody, _ := ioutil.ReadAll(resp.Body)
		errobj := &ErrorResponse{}
		if err = json.Unmarshal(errbody, errobj); err != nil {
			// 如果响应内容非可解析，则返回body原文
			errobj.ErrorMsg = string(errbody)
		}
		return nil, errobj
	}
	return resp.Body, nil
}
