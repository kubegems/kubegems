package oam

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AppRef struct {
	Tenant  string `json:"tenant,omitempty"`
	Project string `json:"project,omitempty"`
	Env     string `json:"environment,omitempty"`

	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type APPFunc func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error)

func (o *OAM) AppRefFunc(req *restful.Request, resp *restful.Response, fun APPFunc) {
	ref := AppRef{
		Tenant:  req.PathParameter("tenant"),
		Project: req.PathParameter("project"),
		Env:     req.PathParameter("environment"),
		Name:    req.PathParameter("name"),
	}
	// check permission
	ctx := req.Request.Context()

	innerfunc := func() (interface{}, error) {
		env := &models.Environment{
			EnvironmentName: ref.Env,
			Project: &models.Project{
				ProjectName: ref.Project,
				Tenant: &models.Tenant{
					TenantName: ref.Tenant,
				},
			},
		}
		if err := o.Database.DB().Preload("Cluster").Take(env).Error; err != nil {
			return nil, err
		}
		clustername, namespace := env.Cluster.ClusterName, env.Namespace
		ref.Namespace = namespace

		cli, err := o.Clientset.ClientOf(ctx, clustername)
		if err != nil {
			return nil, err
		}
		return fun(ctx, cli, ref)
	}

	if data, err := innerfunc(); err != nil {
		response.BadRequest(resp, err.Error())
	} else {
		response.OK(resp, data)
	}
}
