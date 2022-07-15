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

	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/project"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/repository"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
)

type Client struct {
	Ctx       context.Context
	Options   *Options
	ArgoCDcli apiclient.Client

	// cached client
	app     application.ApplicationServiceClient
	repo    repository.RepositoryServiceClient
	cluster cluster.ClusterServiceClient
	project project.ProjectServiceClient
}

func NewClient(ctx context.Context, options *Options) (*Client, error) {
	apiclient, err := NewArgoCDCli(options)
	if err != nil {
		return nil, err
	}
	return &Client{Ctx: ctx, ArgoCDcli: *apiclient, Options: options}, nil
}

func (c *Client) ListArgoApp(ctx context.Context, selector labels.Selector) (*v1alpha1.ApplicationList, error) {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return nil, err
	}

	list, err := cli.List(ctx, &application.ApplicationQuery{Selector: selector.String()})
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) WatchArgoApp(ctx context.Context, name string) (application.ApplicationService_WatchClient, error) {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return nil, err
	}
	return cli.Watch(ctx, &application.ApplicationQuery{Name: &name})
}

func (c *Client) GetArgoApp(ctx context.Context, name string) (*v1alpha1.Application, error) {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return nil, err
	}

	app, err := cli.Get(ctx, &application.ApplicationQuery{Name: &name})
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (c *Client) UpdateApp(ctx context.Context, app *v1alpha1.Application) (*v1alpha1.Application, error) {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return nil, err
	}
	updated, err := cli.Update(ctx, &application.ApplicationUpdateRequest{Application: app})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (c *Client) RemoveArgoApp(ctx context.Context, name string) error {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return err
	}
	_, err = cli.Delete(ctx, &application.ApplicationDeleteRequest{
		Name:    &name,
		Cascade: pointer.Bool(true),
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateArgoApp(ctx context.Context, app *v1alpha1.Application) (*v1alpha1.Application, error) {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return nil, err
	}
	return cli.Create(ctx, &application.ApplicationCreateRequest{Application: *app, Validate: pointer.Bool(false)})
}

func (c *Client) EnsureArgoApp(ctx context.Context, app *v1alpha1.Application) (*v1alpha1.Application, error) {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return nil, err
	}
	return cli.Create(ctx, &application.ApplicationCreateRequest{
		Application: *app,
		Upsert:      pointer.Bool(true),
		Validate:    pointer.Bool(false),
	})
}

func (c *Client) Sync(ctx context.Context, name string, resources []v1alpha1.SyncOperationResource) error {
	appcli, err := c.getAppcli(ctx)
	if err != nil {
		return err
	}
	_, err = appcli.Sync(ctx, &application.ApplicationSyncRequest{
		Name:      &name,
		Resources: resources,
		Strategy:  &v1alpha1.SyncStrategy{Apply: &v1alpha1.SyncStrategyApply{Force: true}},
		Prune:     true,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) RevMeta(ctx context.Context, appname string, rev string) (*v1alpha1.RevisionMetadata, error) {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return nil, err
	}
	return cli.RevisionMetadata(ctx, &application.RevisionMetadataQuery{Name: &appname, Revision: &rev})
}

func (c *Client) EnsureCluster(ctx context.Context, in *v1alpha1.Cluster) (*v1alpha1.Cluster, error) {
	cli, err := c.getclustercli(ctx)
	if err != nil {
		return nil, err
	}

	return cli.Create(ctx, &cluster.ClusterCreateRequest{
		Cluster: in,
		Upsert:  true,
	})
}

func (c *Client) EnsureArgoProject(ctx context.Context, in *v1alpha1.AppProject) (*v1alpha1.AppProject, error) {
	cli, err := c.getprojectcli(ctx)
	if err != nil {
		return nil, err
	}
	return cli.Create(ctx, &project.ProjectCreateRequest{
		Project: in,
		Upsert:  true,
	})
}

func (c *Client) EnsureRepository(ctx context.Context, repo *v1alpha1.Repository) (*v1alpha1.Repository, error) {
	cli, err := c.getRepocli(ctx)
	if err != nil {
		return nil, err
	}
	return cli.Create(ctx, &repository.RepoCreateRequest{Repo: repo, Upsert: true}) // create
}

func (c *Client) ResourceTree(ctx context.Context, name string) (*v1alpha1.ApplicationTree, error) {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return nil, err
	}

	return cli.ResourceTree(ctx, &application.ResourcesQuery{ApplicationName: &name})
}

func (c *Client) WatchResourceTree(ctx context.Context, name string) (application.ApplicationService_WatchResourceTreeClient, error) {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return nil, err
	}
	return cli.WatchResourceTree(ctx, &application.ResourcesQuery{ApplicationName: &name})
}

func (c *Client) DiffResources(ctx context.Context, q *application.ResourcesQuery) ([]*v1alpha1.ResourceDiff, error) {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return nil, err
	}
	data, err := cli.ManagedResources(ctx, q)
	if err != nil {
		return nil, err
	}
	return data.Items, nil
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
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return "", err
	}

	manifest, err := cli.GetResource(ctx, &application.ApplicationResourceRequest{
		Name:         q.Name,
		Namespace:    q.Namespace,
		ResourceName: q.ResourceName,
		Version:      q.Version,
		Group:        q.Group,
		Kind:         q.Kind,
	})
	if err != nil {
		return "", err
	}
	return manifest.Manifest, nil
}

func (c *Client) RemoveResource(ctx context.Context, q ResourceRequest) error {
	cli, err := c.getAppcli(ctx)
	if err != nil {
		return err
	}

	if _, err := cli.DeleteResource(ctx, &application.ApplicationResourceDeleteRequest{
		Name:         q.Name,
		Namespace:    q.Namespace,
		ResourceName: q.ResourceName,
		Version:      q.Version,
		Group:        q.Group,
		Kind:         q.Kind,
	}); err != nil {
		return err
	}
	return nil
}

func (c *Client) getAppcli(ctx context.Context) (application.ApplicationServiceClient, error) {
	// from cache
	if c.app != nil {
		return c.app, nil
	}
	// init application cli
	closer, appcli, err := c.ArgoCDcli.NewApplicationClient()
	if err != nil {
		return nil, err
	}
	go func() {
		<-c.Ctx.Done()
		_ = closer.Close()
	}()
	c.app = appcli
	return appcli, nil
}

func (c *Client) getRepocli(ctx context.Context) (repository.RepositoryServiceClient, error) {
	// from cache
	if c.repo != nil {
		return c.repo, nil
	}
	// init application cli
	closer, repocli, err := c.ArgoCDcli.NewRepoClient()
	if err != nil {
		return nil, err
	}
	go func() {
		<-c.Ctx.Done()
		_ = closer.Close()
	}()
	c.repo = repocli
	return repocli, nil
}

func (c *Client) getclustercli(ctx context.Context) (cluster.ClusterServiceClient, error) {
	// from cache
	if c.cluster != nil {
		return c.cluster, nil
	}
	// init application cli
	closer, clustercli, err := c.ArgoCDcli.NewClusterClient()
	if err != nil {
		return nil, err
	}
	go func() {
		<-c.Ctx.Done()
		_ = closer.Close()
	}()
	c.cluster = clustercli
	return clustercli, nil
}

func (c *Client) getprojectcli(ctx context.Context) (project.ProjectServiceClient, error) {
	// from cache
	if c.project != nil {
		return c.project, nil
	}
	// init application cli
	closer, projectcli, err := c.ArgoCDcli.NewProjectClient()
	if err != nil {
		return nil, err
	}
	go func() {
		<-c.Ctx.Done()
		_ = closer.Close()
	}()
	c.project = projectcli
	return projectcli, nil
}
