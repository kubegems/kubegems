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

package modeldeployments

import (
	"context"
	"crypto"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-logr/logr"
	machinelearningv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/apis/application"
	modelscommon "kubegems.io/kubegems/pkg/apis/models"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
	"kubegems.io/kubegems/pkg/model/deployment"
	storemodels "kubegems.io/kubegems/pkg/model/store/api/models"
	"kubegems.io/kubegems/pkg/model/store/repository"
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

func HashModelName(modelname string) string {
	sha256 := crypto.SHA256.New()
	sha256.Write([]byte(modelname))
	return strings.ToLower(hex.EncodeToString(sha256.Sum(nil)))[:16]
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
			modelscommon.LabelModelSource:   source,
			modelscommon.LabelModelNameHash: HashModelName(modelname),
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
				URL:               md.Status.URL,
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
		paged := response.NewTypedPage(list.Items, listopt.Page, listopt.Size, func(a modelsv1beta1.ModelDeployment) bool {
			return strings.Contains(a.Name, listopt.Search)
		}, func(a, b modelsv1beta1.ModelDeployment) bool {
			return a.CreationTimestamp.After(b.CreationTimestamp.Time)
		})
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

		if err := o.completeModelDeployment(ctx, md, ref); err != nil {
			return nil, err
		}

		// override the namespace
		md.Namespace = ref.Namespace
		if err := cli.Create(ctx, md); err != nil {
			return nil, err
		}
		return md, nil
	})
}

func (o *ModelDeploymentAPI) completeModelDeployment(ctx context.Context, md *modelsv1beta1.ModelDeployment, ref AppRef) error {
	// set labels for selection from model name
	if md.Labels == nil {
		md.Labels = make(map[string]string)
	}
	md.Labels[modelscommon.LabelModelNameHash] = HashModelName(md.Spec.Model.Name)
	md.Labels[modelscommon.LabelModelSource] = md.Spec.Model.Source
	if md.Annotations == nil {
		md.Annotations = make(map[string]string)
	}
	md.Annotations[application.AnnotationRef] = ref.Json()

	// according to the model and source complete the model deployment parameters
	if err := o.completeMDSpec(ctx, md); err != nil {
		return err
	}
	return nil
}

func (o *ModelDeploymentAPI) completeMDSpec(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	// set default gateway
	if md.Spec.Ingress.GatewayName == "" {
		md.Spec.Ingress.GatewayName = "default-gateway"
	}
	source, modelname := md.Spec.Model.Source, md.Spec.Model.Name
	if source == "" || modelname == "" {
		return nil
	}
	sourcedetails, err := o.SourceRepository.Get(ctx, source, repository.GetSourceOptions{})
	if err != nil {
		return err
	}
	modeldetails, err := o.ModelRepository.Get(ctx, source, modelname, false)
	if err != nil {
		return err
	}

	// set task
	md.Spec.Model.Task = modeldetails.Task

	// set first source image if not set
	switch sourcedetails.Kind {
	case repository.SourceKindHuggingface:
		md.Spec.Server.Kind = machinelearningv1.PrepackHuggingFaceName
		md.Spec.Server.Protocol = string(machinelearningv1.ProtocolV2)
		md.Spec.Server.Parameters = append(md.Spec.Server.Parameters,
			modelsv1beta1.Parameter{Name: "task", Value: modeldetails.Task},
			modelsv1beta1.Parameter{Name: "pretrained_model", Value: modeldetails.Name},
		)

		if md.Spec.Server.PodSpec == nil {
			md.Spec.Server.PodSpec = &corev1.PodSpec{}
		}
		// nolint: gomnd
		updatedpodspec := deployment.CreateOrUpdateContainer(*md.Spec.Server.PodSpec, "model", func(c *v1.Container) {
			c.ReadinessProbe = &v1.Probe{InitialDelaySeconds: 120, FailureThreshold: 5}
		})
		md.Spec.Server.PodSpec = &updatedpodspec
	case repository.SourceKindOpenMMLab:
		md.Spec.Server.Kind = modelsv1beta1.PrepackOpenMMLabName
		md.Spec.Server.Protocol = string(machinelearningv1.ProtocolV2)
		md.Spec.Server.Parameters = append(md.Spec.Server.Parameters,
			modelsv1beta1.Parameter{Name: "pkg", Value: modeldetails.Framework},
			modelsv1beta1.Parameter{Name: "model", Value: modeldetails.Name},
		)

		md.Spec.Server.Privileged = true
	case repository.SourceKindModelx:
		md.Spec.Server.Privileged = true
		md.Spec.Server.StorageInitializerImage = "docker.io/kubegems/modelx-dl:latest"
		if md.Spec.Server.StorageInitializerImage == "" {
			md.Spec.Server.StorageInitializerImage = sourcedetails.InitImage
		}
		if md.Spec.Model.URL == "" {
			md.Spec.Model.URL = fmt.Sprintf("%s/%s@%s", sourcedetails.Address, modelname, md.Spec.Model.Version)
		}
	}

	// resource request
	if len(md.Spec.Server.Resources.Requests) == 0 {
		requests := md.Spec.Server.Resources.Limits.DeepCopy()

		requests[corev1.ResourceCPU] = resource.MustParse("100m")
		requests[corev1.ResourceMemory] = resource.MustParse("100Mi")

		md.Spec.Server.Resources.Requests = requests
	}
	return nil
}

func (o *ModelDeploymentAPI) UpdateModelDeployment(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		md := &modelsv1beta1.ModelDeployment{}
		if err := req.ReadEntity(md); err != nil {
			return nil, err
		}
		exist := &modelsv1beta1.ModelDeployment{}
		if err := cli.Get(ctx, client.ObjectKey{Namespace: ref.Namespace, Name: ref.Name}, exist); err != nil {
			return nil, err
		}
		// update fileds
		exist.Spec = md.Spec
		exist.Annotations = md.Annotations
		exist.Labels = md.Labels
		exist.OwnerReferences = md.OwnerReferences

		if err := cli.Update(ctx, exist); err != nil {
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
