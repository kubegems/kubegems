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
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pluginscommon "kubegems.io/kubegems/pkg/apis/plugins"
	"kubegems.io/kubegems/pkg/installer/bundle/helm"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const PluginRepositoriesNamePrefix = "plugin-repository-"

type Repository struct {
	Name     string                     `json:"name,omitempty"`
	Priority int                        `json:"priority,omitempty"`
	Address  string                     `json:"address,omitempty"`
	Plugins  map[string][]PluginVersion `json:"plugins,omitempty"`
	LastSync time.Time                  `json:"lastSync,omitempty"`
}

func (repository *Repository) RefreshRepoIndex(ctx context.Context) error {
	indexFile, err := helm.LoadIndex(ctx, repository.Address)
	if err != nil {
		return err
	}
	pluginversions := map[string][]PluginVersion{}
	for name, chartversions := range indexFile.Entries {
		pvs := make([]PluginVersion, 0, len(chartversions))
		for _, cv := range chartversions {
			if !IsPluginChart(cv) {
				continue
			}
			pvs = append(pvs, PluginVersionFromRepoChartVersion(repository.Address, cv))
		}
		if len(pvs) != 0 {
			pluginversions[name] = pvs
		}
	}
	repository.Plugins = pluginversions
	repository.LastSync = time.Now()
	return nil
}

func (p *PluginManager) DeleteRepo(ctx context.Context, name string) error {
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      PluginRepositoriesNamePrefix + name,
			Namespace: pluginscommon.KubeGemsNamespaceInstaller,
		},
	}
	if err := p.Client.Delete(ctx, secret); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

var isPluginSecretLabel = map[string]string{
	pluginscommon.LabelIsPluginRepo: "true",
}

func (p *PluginManager) GetRepo(ctx context.Context, name string) (*Repository, error) {
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      PluginRepositoriesNamePrefix + name,
			Namespace: pluginscommon.KubeGemsNamespaceInstaller,
		},
	}
	if err := p.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		return nil, err
	}
	repo := repoFromSecret(*secret)
	return &repo, nil
}

func (p *PluginManager) ListRepos(ctx context.Context) (map[string]Repository, error) {
	secretlist := &corev1.SecretList{}
	err := p.Client.List(ctx, secretlist, client.InNamespace(pluginscommon.KubeGemsNamespaceInstaller), client.MatchingLabels(isPluginSecretLabel))
	if err != nil {
		return nil, err
	}
	repos := map[string]Repository{}
	for _, secret := range secretlist.Items {
		repo := repoFromSecret(secret)
		repos[repo.Name] = repo
	}

	return repos, nil
}

func (p *PluginManager) UpdateReposCache(ctx context.Context) error {
	repos, err := p.ListRepos(ctx)
	if err != nil {
		return err
	}
	eg := errgroup.Group{}
	for _, repo := range repos {
		repo := repo
		eg.Go(func() error {
			return p.SetRepo(ctx, &repo, true)
		})
	}
	return eg.Wait()
}

func repoFromSecret(secret corev1.Secret) Repository {
	plugins := map[string][]PluginVersion{}
	_ = json.Unmarshal(secret.Data["plugins"], &plugins)
	lastsync, _ := time.Parse(time.RFC3339, string(secret.Data["lastSync"]))
	priority, _ := strconv.Atoi(string(secret.Data["priority"]))
	return Repository{
		Name:     strings.TrimPrefix(secret.GetName(), PluginRepositoriesNamePrefix),
		Address:  string(secret.Data["address"]),
		Plugins:  plugins,
		LastSync: lastsync,
		Priority: priority,
	}
}

func (p *PluginManager) SetRepo(ctx context.Context, repo *Repository, withRefresh bool) error {
	if withRefresh {
		if err := repo.RefreshRepoIndex(ctx); err != nil {
			return err
		}
	}
	reposecret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      PluginRepositoriesNamePrefix + repo.Name,
			Namespace: pluginscommon.KubeGemsNamespaceInstaller,
			Labels:    isPluginSecretLabel,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, p.Client, reposecret, func() error {
		pluginsraw, err := json.Marshal(repo.Plugins)
		if err != nil {
			return err
		}
		reposecret.Data["plugins"] = pluginsraw
		reposecret.Data["address"] = []byte(repo.Address)
		reposecret.Data["lastSync"] = []byte(repo.LastSync.String())
		return nil
	})
	return err
}
