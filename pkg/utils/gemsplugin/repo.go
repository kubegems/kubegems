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

package gemsplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pluginscommon "kubegems.io/kubegems/pkg/apis/plugins"
	"kubegems.io/kubegems/pkg/installer/bundle/helm"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const PluginRepositoriesName = "plugin-repositories"

type Repository struct {
	Name     string          `json:"name,omitempty"`
	Address  string          `json:"address,omitempty"`
	Index    *repo.IndexFile `json:"index,omitempty"`
	LastSync time.Time       `json:"lastSync,omitempty"`
}

func (r Repository) GetPluginVersions(name string) []PluginVersion {
	for chartname, chartversions := range r.Index.Entries {
		if chartname != name {
			continue
		}
		var pvs []PluginVersion
		for _, cv := range chartversions {
			if !IsPluginChart(cv) {
				continue
			}
			pvs = append(pvs, PluginVersionFromRepoChartVersion(r.Address, cv))
		}
		return pvs
	}
	return nil
}

func (r Repository) ListPluginVersions() map[string][]PluginVersion {
	ret := map[string][]PluginVersion{}
	for name, chartversions := range r.Index.Entries {
		pvs := make([]PluginVersion, 0, len(chartversions))
		for _, cv := range chartversions {
			if !IsPluginChart(cv) {
				continue
			}
			pvs = append(pvs, PluginVersionFromRepoChartVersion(r.Address, cv))
		}
		if len(pvs) != 0 {
			ret[name] = pvs
		}
	}
	return ret
}

func (repository *Repository) RefreshRepoIndex(ctx context.Context) error {
	indexFile, err := helm.LoadRemoteIndex(ctx, repository.Address)
	if err != nil {
		return err
	}
	repository.Index = indexFile
	repository.LastSync = time.Now()
	return nil
}

func (m *PluginManager) SetRepo(ctx context.Context, repository Repository) error {
	if err := repository.RefreshRepoIndex(ctx); err != nil {
		return err
	}
	return m.onSecret(ctx, func(kvs map[string]Repository) error {
		kvs[repository.Name] = repository
		return nil
	})
}

func (m *PluginManager) GetRepo(ctx context.Context, name string, refresh bool) (*Repository, error) {
	var ret *Repository
	if err := m.onSecret(ctx, func(kvs map[string]Repository) error {
		if repo, ok := kvs[name]; ok {
			if refresh {
				repo.RefreshRepoIndex(ctx)
				kvs[name] = repo
			}
			ret = &repo
			return nil
		} else {
			return fmt.Errorf("repo %s not found", name)
		}
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (m *PluginManager) ListRepo(ctx context.Context) ([]Repository, error) {
	ret := []Repository{}
	if err := m.onSecret(ctx, func(kvs map[string]Repository) error {
		for _, v := range kvs {
			ret = append(ret, v)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (m *PluginManager) DeleteRepo(ctx context.Context, name string) error {
	return m.onSecret(ctx, func(kvs map[string]Repository) error {
		for k := range kvs {
			if k == name {
				delete(kvs, k)
			}
		}
		return nil
	})
}

func (m *PluginManager) onSecret(ctx context.Context, fun func(kvs map[string]Repository) error) error {
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      PluginRepositoriesName,
			Namespace: pluginscommon.KubeGemsNamespaceInstaller,
		},
	}
	if err := m.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// add init repo
		initrepo := Repository{Name: "kubegems", Address: KubegemsChartsRepoURL}
		initrepobytes, _ := json.Marshal(initrepo)
		secret.Data = map[string][]byte{initrepo.Name: initrepobytes}
		if err := m.Client.Create(ctx, secret); err != nil {
			return err
		}
	}
	updated := secret.DeepCopy()

	{
		repositories := map[string]Repository{}
		for _, v := range updated.Data {
			repo := &Repository{}
			if err := json.Unmarshal(v, repo); err != nil {
				continue
			}
			repositories[repo.Name] = *repo
		}
		if err := fun(repositories); err != nil {
			return err
		}

		kvs := map[string][]byte{}
		for k, repo := range repositories {
			rawbytes, err := json.Marshal(repo)
			if err != nil {
				continue
			}
			kvs[k] = rawbytes
		}
		updated.Data = kvs
	}

	if equality.Semantic.DeepEqual(secret, updated) {
		return nil
	}
	return m.Client.Patch(ctx, updated, client.MergeFrom(secret))
}
