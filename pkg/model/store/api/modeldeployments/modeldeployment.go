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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-logr/logr"
	machinelearningv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"kubegems.io/kubegems/pkg/apis/application"
	"kubegems.io/kubegems/pkg/apis/models"
	modelscommon "kubegems.io/kubegems/pkg/apis/models"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
	"kubegems.io/kubegems/pkg/model/deployment"
	storemodels "kubegems.io/kubegems/pkg/model/store/api/models"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/library/rest/response"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ModelDeploymentOverview struct {
	AppRef
	Name              string      `json:"name"`
	ModelName         string      `json:"modelName"`
	ModelVersion      string      `json:"modelVersion"`
	URL               string      `json:"url"`
	GRPCAddress       string      `json:"grpcAddress"`
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
	retlist, listmu := []ModelDeploymentOverview{}, &sync.Mutex{}
	eg := errgroup.Group{}
	for _, cluster := range o.Clientset.Clusters() {
		cluster := cluster
		eg.Go(func() error {
			cli, err := o.Clientset.ClientOf(ctx, cluster)
			if err != nil {
				log.Error(err, "failed to get client for cluster", "cluster", cluster)
				return err
			}
			list := &modelsv1beta1.ModelDeploymentList{}
			// nolint: gomnd
			ctx, cancel := context.WithTimeout(ctx, time.Second*5)
			defer cancel()

			if err := cli.List(ctx, list, client.MatchingLabels{
				modelscommon.LabelModelSource:   source,
				modelscommon.LabelModelNameHash: HashModelName(modelname),
			}); err != nil {
				log.Error(err, "failed to list model deployments", "cluster", cluster)
				return err
			}
			listmu.Lock()
			defer listmu.Unlock()
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
					GRPCAddress:       md.Status.GRPCAddress,
					Phase:             string(md.Status.Phase),
					Cluster:           cluster,
					Namespace:         md.Namespace,
					Creator:           appref.Username,
					AppRef:            *appref,
					CreationTimestamp: md.CreationTimestamp,
				})
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		log.Error(err, "wait list model deployments")
	}
	// sort by creation timestamp desc
	getname := func(i ModelDeploymentOverview) string { return i.Name }
	gettime := func(i ModelDeploymentOverview) time.Time { return i.CreationTimestamp.Time }
	paged := response.PageFromRequest(req.Request, retlist, getname, gettime)
	response.OK(resp, paged)
}

func (o *ModelDeploymentAPI) ListModelDeployments(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		list := &modelsv1beta1.ModelDeploymentList{}
		if err := cli.List(ctx, list, client.InNamespace(ref.Namespace)); err != nil {
			return nil, err
		}
		paged := response.PageObjectFromRequest(req.Request, list.Items)
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
	// set default gateway
	if md.Spec.Ingress.GatewayName == "" {
		md.Spec.Ingress.GatewayName = "default-gateway"
	}
	// set first source image if not set
	switch sourcedetails.Kind {
	case repository.SourceKindHuggingface:
		md.Spec.Server.Kind = machinelearningv1.PrepackHuggingFaceName
		md.Spec.Server.Protocol = string(machinelearningv1.ProtocolV2)
		md.Spec.Server.Parameters = append(md.Spec.Server.Parameters,
			modelsv1beta1.Parameter{Name: "task", Value: modeldetails.Task},
			modelsv1beta1.Parameter{Name: "pretrained_model", Value: modeldetails.Name},
		)
	case repository.SourceKindOpenMMLab:
		md.Spec.Server.Kind = modelsv1beta1.PrepackOpenMMLabName
		md.Spec.Server.Protocol = string(machinelearningv1.ProtocolV2)
		md.Spec.Server.Parameters = append(md.Spec.Server.Parameters,
			modelsv1beta1.Parameter{Name: "pkg", Value: modeldetails.Framework},
			modelsv1beta1.Parameter{Name: "model", Value: modeldetails.Name},
		)
		md.Spec.Server.Privileged = true
	case repository.SourceKindModelx:
		md.Spec.Server.Kind = modelsv1beta1.ServerKindModelx
		md.Spec.Model.URL = sourcedetails.Address
		md.Spec.Server.Privileged = true
		md.Spec.Server.StorageInitializerImage = o.ModelxStorageInitalizer
		if md.Spec.Server.StorageInitializerImage == "" {
			md.Spec.Server.StorageInitializerImage = sourcedetails.InitImage
		}
	}
	completeProbes(md)
	// resource request
	if len(md.Spec.Server.Resources.Requests) == 0 {
		requests := md.Spec.Server.Resources.Limits.DeepCopy()
		requests[corev1.ResourceCPU] = resource.MustParse("100m")
		requests[corev1.ResourceMemory] = resource.MustParse("100Mi")
		md.Spec.Server.Resources.Requests = requests
	}
	removeEmptyResource(md.Spec.Server.Resources.Requests)
	removeEmptyResource(md.Spec.Server.Resources.Limits)
	return nil
}

func removeEmptyResource(list corev1.ResourceList) {
	for k, v := range list {
		if v.IsZero() {
			delete(list, k)
		}
	}
}

const (
	DefaultInitDeplaySeconds = 180
	ModelxInitDeplaySeconds  = 60
)

func completeProbes(md *modelsv1beta1.ModelDeployment) {
	if len(md.Spec.Server.Ports) == 0 && md.Spec.Server.Kind == modelsv1beta1.ServerKindModelx {
		return
	}
	if enabled, _ := strconv.ParseBool(md.Annotations[models.AnnotationEnableProbes]); !enabled {
		return
	}
	initDelay := DefaultInitDeplaySeconds
	if md.Spec.Server.Kind == modelsv1beta1.ServerKindModelx {
		initDelay = ModelxInitDeplaySeconds
	}
	md.Spec.Server.PodSpec = deployment.CreateOrUpdateContainer(md.Spec.Server.PodSpec, deployment.ModelContainerName, func(c *v1.Container, _ *v1.PodSpec) {
		// nolint: gomnd
		probe := &v1.Probe{
			InitialDelaySeconds: int32(initDelay),
			PeriodSeconds:       10,
			FailureThreshold:    12,
		}
		// seldon'll set probe handler to selon's default when using MLServer
		// but no server will not.
		if md.Spec.Server.Kind == modelsv1beta1.ServerKindModelx {
			probe.ProbeHandler = v1.ProbeHandler{TCPSocket: &v1.TCPSocketAction{
				Port: detectMainPort(append(md.Spec.Server.Ports, c.Ports...)),
			}}
		}
		if c.ReadinessProbe == nil {
			c.ReadinessProbe = probe
		}
		if c.LivenessProbe == nil {
			c.LivenessProbe = probe
		}
	})
}

func detectMainPort(ports []v1.ContainerPort) intstr.IntOrString {
	if len(ports) == 0 {
		return intstr.FromString("")
	}
	// find http prefix first
	var findport *v1.ContainerPort
	for i, port := range ports {
		if strings.Contains(port.Name, "http") {
			findport = &ports[i]
			break
		}
	}
	if findport == nil {
		findport = &ports[0]
	}
	if findport.Name != "" {
		return intstr.FromString(findport.Name)
	}
	return intstr.FromInt(int(findport.ContainerPort))
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
