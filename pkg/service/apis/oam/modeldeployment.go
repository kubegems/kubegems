package oam

import (
	"context"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/apis/application"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ModelDeploymentOverview struct {
	Name      string `json:"name"`
	ModelName string `json:"modelName"`
	URL       string `json:"url"`
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
	Creator   string `json:"creator"`

	Tenant  string `json:"tenant"`
	Project string `json:"project"`
	Env     string `json:"env"`
}

func (o *OAM) ListAllModelDeployments(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	log := logr.FromContextOrDiscard(ctx).WithValues("method", "ListAllModelDeployments")

	retlist := []ModelDeploymentOverview{}
	for _, cluster := range o.Clientset.Clusters() {
		cli, err := o.Clientset.ClientOf(ctx, cluster)
		if err != nil {
			log.Error(err, "failed to get client for cluster", "cluster", cluster)
			continue
		}
		list := &modelsv1beta1.ModelDeploymentList{}
		if err := cli.List(ctx, list); err != nil {
			log.Error(err, "failed to list model deployments", "cluster", cluster)
			continue
		}
		for _, md := range list.Items {
			if md.Annotations == nil {
				md.Annotations = make(map[string]string)
			}
			retlist = append(retlist, ModelDeploymentOverview{
				Name:      md.Name,
				ModelName: md.Spec.Model.Name,
				URL:       md.Spec.Host,
				Cluster:   cluster,
				Namespace: md.Namespace,
				Creator:   md.Annotations[application.AnnotationCreator],
			})
		}
	}
	response.OK(resp, retlist)
}

func (o *OAM) ListModelDeployments(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		list := &modelsv1beta1.ModelDeploymentList{}
		if err := cli.List(ctx, list, client.InNamespace(ref.Namespace)); err != nil {
			return nil, err
		}
		listopt := request.GetListOptions(req.Request)
		paged := response.NewPageData(list.Items, listopt.Page, listopt.Size, func(i int) bool {
			return strings.Contains(list.Items[i].Name, listopt.Search)
		}, nil)
		return paged, nil
	})
}

func (o *OAM) GetModelDeployment(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		md := &modelsv1beta1.ModelDeployment{}
		if err := cli.Get(ctx, client.ObjectKey{Name: ref.Name, Namespace: ref.Namespace}, md); err != nil {
			return nil, err
		}
		return md, nil
	})
}

func (o *OAM) CreateModelDeployment(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		md := &modelsv1beta1.ModelDeployment{}
		if err := req.ReadEntity(md); err != nil {
			return nil, err
		}
		// set the namespace
		md.Namespace = ref.Namespace
		if err := cli.Create(ctx, md); err != nil {
			return nil, err
		}
		return md, nil
	})
}

func (o *OAM) UpdateModelDeployment(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		md := &modelsv1beta1.ModelDeployment{}
		if err := req.ReadEntity(md); err != nil {
			return nil, err
		}
		// set the namespace
		md.Namespace = ref.Namespace
		if err := cli.Update(ctx, md); err != nil {
			return nil, err
		}
		return md, nil
	})
}

func (o *OAM) DeleteModelDeployment(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		md := &modelsv1beta1.ModelDeployment{ObjectMeta: metav1.ObjectMeta{Name: ref.Name, Namespace: ref.Namespace}}
		if err := cli.Delete(ctx, md); err != nil {
			return nil, err
		}
		return md, nil
	})
}

func (o *OAM) PatchModelDeployment(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		md := &modelsv1beta1.ModelDeployment{}
		if err := req.ReadEntity(md); err != nil {
			return nil, err
		}
		md.Namespace = ref.Namespace
		if err := cli.Patch(ctx, md, client.MergeFrom(md)); err != nil {
			return nil, err
		}
		return md, nil
	})
}
