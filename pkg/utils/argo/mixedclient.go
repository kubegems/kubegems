package argo

import (
	"context"
	"net/url"
	"strings"

	argocommon "github.com/argoproj/argo-cd/v2/common"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/project"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/repository"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/kubegems/gems/pkg/utils/agents"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	Ctx       context.Context
	Options   *Options
	ArgoCDcli apiclient.Client
	AgentsCli *agents.ClientSet
	// cached client
	app        application.ApplicationServiceClient
	repo       repository.RepositoryServiceClient
	cluster    cluster.ClusterServiceClient
	project    project.ProjectServiceClient
	managercli *agents.TypedClient
}

func NewClient(ctx context.Context, options *Options, agentsClientSet *agents.ClientSet) (*Client, error) {
	apiclient, err := NewArgoCDCli(options)
	if err != nil {
		return nil, err
	}
	return &Client{
		Ctx:       ctx,
		ArgoCDcli: *apiclient,
		AgentsCli: agentsClientSet,
		Options:   options,
	}, nil
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

func (c *Client) WatchAppK8s(ctx context.Context, name string) (watch.Interface, error) {
	cli, err := c.getmanagerCli(ctx)
	if err != nil {
		return nil, err
	}
	applist := &v1alpha1.ApplicationList{}
	return cli.Watch(ctx, applist, client.InNamespace(c.Options.Namespace))
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
	created := app.DeepCopy()
	created.Namespace = c.Options.Namespace
	return cli.Create(ctx, &application.ApplicationCreateRequest{Application: *app, Validate: pointer.Bool(false)})
}

func (c *Client) CreateArgoAppK8s(ctx context.Context, app *v1alpha1.Application) (*v1alpha1.Application, error) {
	cli, err := c.getmanagerCli(ctx)
	if err != nil {
		return nil, err
	}
	created := app.DeepCopy()
	created.Namespace = c.Options.Namespace
	if err := cli.Create(ctx, created); err != nil {
		return nil, err
	}
	return created, nil
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

// https://github.com/argoproj/argo-cd/blob/v2.1.7/server/application/application.go#L1552
// SyncK8s 触发同步，如果有resources指定则仅同步指定的，否则同步全部。
func (c *Client) SyncK8s(ctx context.Context, name string, resources []v1alpha1.SyncOperationResource) error {
	cli, err := c.getmanagerCli(ctx)
	if err != nil {
		// backoff
		return c.Sync(ctx, name, resources)
	}

	app := &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Options.Namespace,
		},
	}
	changed := app.DeepCopy()
	changed.Status.OperationState = nil
	changed.Operation = &v1alpha1.Operation{
		Sync: &v1alpha1.SyncOperation{
			Resources: resources,
			Prune:     true, // 清理多余的资源
			SyncStrategy: &v1alpha1.SyncStrategy{
				Apply: &v1alpha1.SyncStrategyApply{
					Force: true, // 有冲突时覆盖
				},
			},
		},
	}
	patch := client.MergeFrom(app)
	return cli.Patch(ctx, changed, patch)
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

// https://github.com/argoproj/argo-cd/blob/v2.1.7/server/application/application.go#L1346
func (c *Client) TerminalOperation(ctx context.Context, app *v1alpha1.Application) error {
	cli, err := c.getmanagerCli(ctx)
	if err != nil {
		return err
	}
	mergepatch := `{"status":{"operationState":{"phase":"Terminating"}}}`
	patch := client.RawPatch(types.MergePatchType, []byte(mergepatch))
	return cli.Patch(ctx, app, patch)
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

func (c *Client) EnsureRepositoryK8s(ctx context.Context, repo *v1alpha1.Repository) (*v1alpha1.Repository, error) {
	cli, err := c.getmanagerCli(ctx)
	if err != nil {
		return nil, err
	}

	// bug: 查询不存在的 repository 返回如下，且 error 为空
	/*
		*github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1.Repository
			{
		    Repo: "",
		    Username: "",
		    Password: "",
		    SSHPrivateKey: "",
		    ConnectionState: github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1.ConnectionState {
		        Status: "Failed",
		        Message: "Unable to connect to repository: Get \"index.yaml\": unsupported protocol scheme \"\"",
		        ModifiedAt: *(*"k8s.io/apimachinery/pkg/apis/meta/v1.Time")(0xc0028f7a58)
		    },
		    InsecureIgnoreHostKey: false,
		    Insecure: false,
		    EnableLFS: false,
		    TLSClientCertData: "",
		    TLSClientCertKey: "",
		    Type: "git",
		    Name: "",
		    InheritedCreds: false,
		    EnableOCI: false,
		    GithubAppPrivateKey: "",
		    GithubAppId: 0,
		    GithubAppInstallationId: 0,
		    GitHubAppEnterpriseBaseURL: ""
		}
	*/

	// 根据 https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#repositories 先使用k8s方式配置

	repourl, err := url.Parse(repo.Repo)
	if err != nil {
		return nil, err
	}

	replacer := strings.NewReplacer(
		"/", "-",
		"@", "-",
		":", "-",
		".", "-",
	)
	reponame := replacer.Replace(repourl.Host + repourl.Path)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      reponame,
			Namespace: c.Options.Namespace,
			Labels: map[string]string{
				argocommon.LabelKeySecretType: "repository",
			},
		},
		StringData: map[string]string{
			"type":     "git",
			"url":      repo.Repo,
			"password": repo.Password,
			"username": repo.Username,
		},
	}

	exist := &corev1.Secret{}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(secret), exist); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		if err := cli.Create(ctx, secret); err != nil {
			return nil, err
		}
	}
	return repo, nil
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

func (c *Client) getmanagerCli(ctx context.Context) (*agents.TypedClient, error) {
	// from cache
	if c.managercli != nil {
		return c.managercli, nil
	}
	// init managercli
	cli, err := c.AgentsCli.ClientOfManager(ctx)
	if err != nil {
		return nil, err
	}
	managercli := cli.TypedClient
	c.managercli = managercli
	return managercli, nil
}
