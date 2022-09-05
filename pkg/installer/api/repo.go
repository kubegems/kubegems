package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/emicklei/go-restful/v3"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const PluginRepositoriesName = "plugin-repositories"

func (o *PluginsAPI) RepoUpdate(req *restful.Request, resp *restful.Response) {
	reponame := req.PathParameter("name")
	if err := o.manager.RefeshRepo(req.Request.Context(), reponame); err != nil {
		response.Error(resp, err)
	} else {
		response.OK(resp, "ok")
	}
}

func (o *PluginsAPI) RepoList(req *restful.Request, resp *restful.Response) {
	repos, err := o.manager.ListRepo(req.Request.Context())
	if err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, repos)
}

func (o *PluginsAPI) RepoAdd(req *restful.Request, resp *restful.Response) {
	repo := &Repository{}
	if err := request.Body(req.Request, &repo); err != nil {
		response.Error(resp, err)
		return
	}
	if err := o.manager.SetRepo(req.Request.Context(), *repo); err != nil {
		response.Error(resp, err)
		return
	}

	// sync
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		o.manager.RefeshRepo(ctx, repo.Name)
	}()

	response.OK(resp, repo)
}

func (o *PluginsAPI) RepoRemove(req *restful.Request, resp *restful.Response) {
	reponame := req.PathParameter("name")
	if err := o.manager.DeleteRepo(req.Request.Context(), reponame); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, "ok")
}

type Repository struct {
	Name     string          `json:"name,omitempty"`
	Address  string          `json:"address,omitempty"`
	Index    *repo.IndexFile `json:"index,omitempty"`
	LastSync time.Time       `json:"lastSync,omitempty"`
}

func (repository *Repository) RefreshRepoIndex(ctx context.Context) error {
	cli, err := getter.NewHTTPGetter()
	if err != nil {
		return err
	}

	resp, err := cli.Get(repository.Address + "/index.yaml")
	if err != nil {
		return err
	}
	index, err := io.ReadAll(resp)
	if err != nil {
		return err
	}
	indexFile, err := LoadIndex(index)
	if err != nil {
		return err
	}
	repository.Index = indexFile
	repository.LastSync = time.Now()
	return nil
}

func (m *pluginManager) SetRepo(ctx context.Context, repository Repository) error {
	return m.onSecret(ctx, func(kvs map[string]Repository) error {
		kvs[repository.Name] = repository
		return nil
	})
}

func (m *pluginManager) RefeshRepo(ctx context.Context, name string) error {
	return m.onSecret(ctx, func(kvs map[string]Repository) error {
		repository, ok := kvs[name]
		if !ok {
			return fmt.Errorf("repository %s not exists", name)
		}
		if err := repository.RefreshRepoIndex(ctx); err != nil {
			return err
		}
		kvs[name] = repository
		return nil
	})
}

func (m *pluginManager) ListRepo(ctx context.Context) ([]Repository, error) {
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

func (m *pluginManager) DeleteRepo(ctx context.Context, name string) error {
	return m.onSecret(ctx, func(kvs map[string]Repository) error {
		for k := range kvs {
			if k == name {
				delete(kvs, k)
			}
		}
		return nil
	})
}

func (m *pluginManager) onSecret(ctx context.Context, fun func(kvs map[string]Repository) error) error {
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      PluginRepositoriesName,
			Namespace: m.namespace,
		},
	}
	if err := m.cli.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if err := m.cli.Create(ctx, secret); err != nil {
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
	return m.cli.Patch(ctx, updated, client.MergeFrom(secret))
}

// The source parameter is only used for logging.
// This will fail if API Version is not set (ErrNoAPIVersion) or if the unmarshal fails.
func LoadIndex(data []byte) (*repo.IndexFile, error) {
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
