package modeldeployments

import (
	"context"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/apis/application"
	modelscommon "kubegems.io/kubegems/pkg/apis/models"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
	storemodels "kubegems.io/kubegems/pkg/model/store/api/models"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ModelDeploymentOverview struct {
	AppRef
	Name              string      `json:"name"`
	ModelName         string      `json:"modelName"`
	ModelVersion      string      `json:"modelVersion"`
	URL               string      `json:"url"`
	Cluster           string      `json:"cluster"`
	Namespace         string      `json:"namespace"`
	Creator           string      `json:"creator"`
	Phase             string      `json:"phase"`
	CreationTimestamp metav1.Time `json:"creationTimestamp"`
}

func EncodeModelName(modelname string) string {
	// can't use the model name as label value which contains '/'
	return strings.Replace(modelname, "/", ".", -1)
}

func (o *ModelDeploymentAPI) ListAllModelDeployments(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	log := logr.FromContextOrDiscard(ctx).WithValues("method", "ListAllModelDeployments")
	source, modelname := storemodels.DecodeSourceModelName(req)
	retlist := []ModelDeploymentOverview{}
	for _, cluster := range o.Clientset.Clusters() {
		cli, err := o.Clientset.ClientOf(ctx, cluster)
		if err != nil {
			log.Error(err, "failed to get client for cluster", "cluster", cluster)
			continue
		}
		list := &modelsv1beta1.ModelDeploymentList{}
		if err := cli.List(ctx, list, client.MatchingLabels{
			modelscommon.LabelModelSource: source,
			modelscommon.LabelModelName:   EncodeModelName(modelname),
		}); err != nil {
			log.Error(err, "failed to list model deployments", "cluster", cluster)
			continue
		}
		for _, md := range list.Items {
			if md.Annotations == nil {
				md.Annotations = make(map[string]string)
			}

			appref := &AppRef{}
			appref.FromJson(md.Annotations[application.AnnotationRef])
			retlist = append(retlist, ModelDeploymentOverview{
				Name:              md.Name,
				ModelName:         md.Spec.Model.Name,
				ModelVersion:      md.Spec.Model.Version,
				URL:               "http://" + md.Spec.Host,
				Phase:             string(md.Status.Phase),
				Cluster:           cluster,
				Namespace:         md.Namespace,
				Creator:           appref.Username,
				AppRef:            *appref,
				CreationTimestamp: md.CreationTimestamp,
			})
		}
	}

	listoptions := request.GetListOptions(req.Request)
	// sort by creation timestamp desc
	paged := response.NewPageData(retlist, listoptions.Page, listoptions.Size, nil, func(i, j int) bool {
		return retlist[i].CreationTimestamp.After(retlist[j].CreationTimestamp.Time)
	})
	response.OK(resp, paged)
}

func (o *ModelDeploymentAPI) ListModelDeployments(req *restful.Request, resp *restful.Response) {
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

func (o *ModelDeploymentAPI) GetModelDeployment(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		md := &modelsv1beta1.ModelDeployment{}
		if err := cli.Get(ctx, client.ObjectKey{Name: ref.Name, Namespace: ref.Namespace}, md); err != nil {
			return nil, err
		}
		return md, nil
	})
}

func (o *ModelDeploymentAPI) CreateModelDeployment(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		md := &modelsv1beta1.ModelDeployment{}
		if err := req.ReadEntity(md); err != nil {
			return nil, err
		}
		// 因选择模型商店需要展示特定模型的实例，为便于选择，设置模型名称label
		if md.Labels == nil {
			md.Labels = make(map[string]string)
		}

		md.Labels[modelscommon.LabelModelName] = EncodeModelName(md.Spec.Model.Name)
		md.Labels[modelscommon.LabelModelSource] = md.Spec.Model.Source
		// 为便于示租户项目环境，设置annotations
		if md.Annotations == nil {
			md.Annotations = make(map[string]string)
		}
		md.Annotations[application.AnnotationRef] = ref.Json()
		// set the namespace
		md.Namespace = ref.Namespace
		if err := cli.Create(ctx, md); err != nil {
			return nil, err
		}
		return md, nil
	})
}

func (o *ModelDeploymentAPI) UpdateModelDeployment(req *restful.Request, resp *restful.Response) {
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

func (o *ModelDeploymentAPI) DeleteModelDeployment(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		md := &modelsv1beta1.ModelDeployment{ObjectMeta: metav1.ObjectMeta{Name: ref.Name, Namespace: ref.Namespace}}
		if err := cli.Delete(ctx, md); err != nil {
			return nil, err
		}
		return md, nil
	})
}

func (o *ModelDeploymentAPI) PatchModelDeployment(req *restful.Request, resp *restful.Response) {
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
