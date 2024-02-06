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

package argo

import (
	"context"
	"sync"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/project"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/repository"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
)

type clientCache struct {
	apiclient.Client
	// cached client
	close   chan struct{}
	app     application.ApplicationServiceClient
	repo    repository.RepositoryServiceClient
	cluster cluster.ClusterServiceClient
	project project.ProjectServiceClient
}

type Client struct {
	Ctx     context.Context
	lock    sync.Mutex
	Options *Options
	cli     *clientCache
}

func NewLazyClient(ctx context.Context, options *Options) *Client {
	return &Client{Ctx: ctx, Options: options}
}

func NewClient(ctx context.Context, options *Options) (*Client, error) {
	cli := NewLazyClient(ctx, options)
	// init client to validate configuration
	_, err := Argoclifunc(cli, func(c *clientCache) (string, error) {
		return "", nil
	})
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func (c *Client) ListArgoApp(ctx context.Context, selector labels.Selector) (*v1alpha1.ApplicationList, error) {
	return appfunc(c, ctx, func(cli application.ApplicationServiceClient) (*v1alpha1.ApplicationList, error) {
		return cli.List(ctx, &application.ApplicationQuery{Selector: selector.String()})
	})
}

func (c *Client) WatchArgoApp(ctx context.Context, name string) (application.ApplicationService_WatchClient, error) {
	return appfunc(c, ctx, func(cli application.ApplicationServiceClient) (application.ApplicationService_WatchClient, error) {
		return cli.Watch(ctx, &application.ApplicationQuery{Name: &name})
	})
}

func (c *Client) GetArgoApp(ctx context.Context, name string) (*v1alpha1.Application, error) {
	return appfunc(c, ctx, func(cli application.ApplicationServiceClient) (*v1alpha1.Application, error) {
		return cli.Get(ctx, &application.ApplicationQuery{Name: &name})
	})
}

func (c *Client) UpdateApp(ctx context.Context, app *v1alpha1.Application) (*v1alpha1.Application, error) {
	return appfunc(c, ctx, func(cli application.ApplicationServiceClient) (*v1alpha1.Application, error) {
		return cli.Update(ctx, &application.ApplicationUpdateRequest{Application: app})
	})
}

func (c *Client) RemoveArgoApp(ctx context.Context, name string) error {
	_, err := appfunc(c, ctx, func(cli application.ApplicationServiceClient) (*application.ApplicationResponse, error) {
		return cli.Delete(ctx, &application.ApplicationDeleteRequest{
			Name:    &name,
			Cascade: pointer.Bool(true),
		})
	})
	return err
}

func (c *Client) CreateArgoApp(ctx context.Context, app *v1alpha1.Application) (*v1alpha1.Application, error) {
	return appfunc(c, ctx, func(cli application.ApplicationServiceClient) (*v1alpha1.Application, error) {
		return cli.Create(ctx, &application.ApplicationCreateRequest{Application: *app, Validate: pointer.Bool(false)})
	})
}

func (c *Client) EnsureArgoApp(ctx context.Context, app *v1alpha1.Application) (*v1alpha1.Application, error) {
	return appfunc(c, ctx, func(cli application.ApplicationServiceClient) (*v1alpha1.Application, error) {
		return cli.Create(ctx, &application.ApplicationCreateRequest{
			Application: *app,
			Upsert:      pointer.Bool(true),
			Validate:    pointer.Bool(false),
		})
	})
}

func (c *Client) Sync(ctx context.Context, name string, resources []v1alpha1.SyncOperationResource) error {
	_, err := appfunc(c, ctx, func(cli application.ApplicationServiceClient) (*v1alpha1.Application, error) {
		return cli.Sync(ctx, &application.ApplicationSyncRequest{
			Name:      &name,
			Resources: resources,
			Strategy:  &v1alpha1.SyncStrategy{Apply: &v1alpha1.SyncStrategyApply{Force: true}},
			Prune:     true,
		})
	})
	return err
}

func (c *Client) ResourceTree(ctx context.Context, name string) (*v1alpha1.ApplicationTree, error) {
	return appfunc(c, ctx, func(cli application.ApplicationServiceClient) (*v1alpha1.ApplicationTree, error) {
		return cli.ResourceTree(ctx, &application.ResourcesQuery{ApplicationName: &name})
	})
}

func (c *Client) WatchResourceTree(ctx context.Context, name string) (application.ApplicationService_WatchResourceTreeClient, error) {
	return appfunc(c, ctx, func(cli application.ApplicationServiceClient) (application.ApplicationService_WatchResourceTreeClient, error) {
		return cli.WatchResourceTree(ctx, &application.ResourcesQuery{ApplicationName: &name})
	})
}

func (c *Client) DiffResources(ctx context.Context, q *application.ResourcesQuery) ([]*v1alpha1.ResourceDiff, error) {
	ret, err := appfunc(c, ctx, func(cli application.ApplicationServiceClient) (*application.ManagedResourcesResponse, error) {
		return cli.ManagedResources(ctx, q)
	})
	if err != nil {
		return nil, err
	}
	return ret.Items, nil
}

type ResourceRequest struct {
	Name         *string `json:"name,omitempty"`         // application name
	Namespace    string  `json:"namespace,omitempty"`    // resource namespace
	ResourceName string  `json:"resourceName,omitempty"` // resource name
	Version      string  `json:"version,omitempty"`
	Group        string  `json:"group,omitempty"`
	Kind         string  `json:"kind,omitempty"`
}

func (c *Client) GetResource(ctx context.Context, q ResourceRequest) (string, error) {
	manifest, err := appfunc(c, ctx, func(cli application.ApplicationServiceClient) (*application.ApplicationResourceResponse, error) {
		return cli.GetResource(ctx, &application.ApplicationResourceRequest{
			Name:         q.Name,
			Namespace:    q.Namespace,
			ResourceName: q.ResourceName,
			Version:      q.Version,
			Group:        q.Group,
			Kind:         q.Kind,
		})
	})
	if err != nil {
		return "", err
	}
	return manifest.Manifest, nil
}

func (c *Client) RemoveResource(ctx context.Context, q ResourceRequest) error {
	_, err := appfunc(c, ctx, func(cli application.ApplicationServiceClient) (*application.ApplicationResponse, error) {
		return cli.DeleteResource(ctx, &application.ApplicationResourceDeleteRequest{
			Name:         q.Name,
			Namespace:    q.Namespace,
			ResourceName: q.ResourceName,
			Version:      q.Version,
			Group:        q.Group,
			Kind:         q.Kind,
		})
	})
	return err
}

func (c *Client) EnsureCluster(ctx context.Context, in *v1alpha1.Cluster) (*v1alpha1.Cluster, error) {
	return clusterfunc(c, ctx, func(cli cluster.ClusterServiceClient) (*v1alpha1.Cluster, error) {
		return cli.Create(ctx, &cluster.ClusterCreateRequest{Cluster: in, Upsert: true})
	})
}

func (c *Client) EnsureArgoProject(ctx context.Context, in *v1alpha1.AppProject) (*v1alpha1.AppProject, error) {
	return projectfunc(c, ctx, func(cli project.ProjectServiceClient) (*v1alpha1.AppProject, error) {
		return cli.Create(ctx, &project.ProjectCreateRequest{Project: in, Upsert: true})
	})
}

func (c *Client) EnsureRepository(ctx context.Context, repo *v1alpha1.Repository) (*v1alpha1.Repository, error) {
	return repofunc(c, ctx, func(cli repository.RepositoryServiceClient) (*v1alpha1.Repository, error) {
		return cli.Create(ctx, &repository.RepoCreateRequest{Repo: repo, Upsert: true})
	})
}

func appfunc[T any](cli *Client, ctx context.Context, fn func(cli application.ApplicationServiceClient) (T, error)) (T, error) {
	return Argoclifunc(cli, func(c *clientCache) (T, error) {
		if c.app == nil {
			closer, appcli, err := c.NewApplicationClient()
			if err != nil {
				return *new(T), err
			}
			go func() {
				<-c.close
				_ = closer.Close()
			}()
			c.app = appcli
		}
		return fn(c.app)
	})
}

func clusterfunc[T any](cli *Client, ctx context.Context, fn func(cli cluster.ClusterServiceClient) (T, error)) (T, error) {
	return Argoclifunc(cli, func(c *clientCache) (T, error) {
		if c.cluster == nil {
			closer, innercli, err := c.NewClusterClient()
			if err != nil {
				return *new(T), err
			}
			go func() {
				<-c.close
				_ = closer.Close()
			}()
			c.cluster = innercli
		}
		return fn(c.cluster)
	})
}

func projectfunc[T any](cli *Client, ctx context.Context, fn func(cli project.ProjectServiceClient) (T, error)) (T, error) {
	return Argoclifunc(cli, func(c *clientCache) (T, error) {
		if c.project == nil {
			closer, innercli, err := c.NewProjectClient()
			if err != nil {
				return *new(T), err
			}
			go func() {
				<-c.close
				_ = closer.Close()
			}()
			c.project = innercli
		}
		return fn(c.project)
	})
}

func repofunc[T any](cli *Client, ctx context.Context, fn func(cli repository.RepositoryServiceClient) (T, error)) (T, error) {
	return Argoclifunc(cli, func(c *clientCache) (T, error) {
		if c.repo == nil {
			closer, innercli, err := c.NewRepoClient()
			if err != nil {
				return *new(T), err
			}
			go func() {
				<-c.close
				_ = closer.Close()
			}()
			c.repo = innercli
		}
		return fn(c.repo)
	})
}

func Argoclifunc[T any](c *Client, fn func(c *clientCache) (T, error)) (T, error) {
	if c.cli == nil {
		c.lock.Lock()
		defer c.lock.Unlock()
		if c.cli == nil {
			null := *new(T)
			apiclient, err := NewArgoCDCli(c.Options)
			if err != nil {
				return null, err
			}
			c.cli = &clientCache{
				close:  make(chan struct{}),
				Client: apiclient,
			}
		}
	}
	ret, err := fn(c.cli)
	// hook to refresh argo client cache
	if status.Code(err) == codes.Unauthenticated {
		// refresh cli
		c.lock.Lock()
		defer c.lock.Unlock()
		if c.cli != nil {
			close(c.cli.close)
			c.cli = nil
		}
	}
	return ret, err
}
