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
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

func LoadIndex(ctx context.Context, uri string) (*repo.IndexFile, error) {
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "https":
		return LoadRemoteIndex(ctx, uri)
	case "file":
		return LoadLocalIndex(uri)
	default:
		return nil, fmt.Errorf("unsupported uri %s", uri)
	}
}

func LoadLocalIndex(path string) (*repo.IndexFile, error) {
	indexcontent, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadIndexData(indexcontent)
}

func LoadRemoteIndex(ctx context.Context, repo string) (*repo.IndexFile, error) {
	resp, err := HTTPGet(ctx, repo+"/index.yaml")
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	index, err := io.ReadAll(resp)
	if err != nil {
		return nil, err
	}
	indexFile, err := LoadIndexData(index)
	if err != nil {
		return nil, err
	}
	return indexFile, nil
}

// The source parameter is only used for logging.
// This will fail if API Version is not set (ErrNoAPIVersion) or if the unmarshal fails.
func LoadIndexData(data []byte) (*repo.IndexFile, error) {
	i := &repo.IndexFile{}
	if len(data) == 0 {
		return i, repo.ErrEmptyIndexYaml
	}
	if err := yaml.UnmarshalStrict(data, i); err != nil {
		return i, err
	}
	for _, cvs := range i.Entries {
		for idx := len(cvs) - 1; idx >= 0; idx-- {
			if err := cvs[idx].Validate(); err != nil {
				cvs = append(cvs[:idx], cvs[idx+1:]...)
			}
		}
	}
	i.SortEntries()
	return i, nil
}
