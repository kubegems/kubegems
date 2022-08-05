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

package deployment

import (
	"context"
	"net/url"

	machinelearningv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const SeldonModelServeKind = "seldon"

type SeldonModelServe struct {
	Client      client.Client
	IngressHost string
}

func (r *SeldonModelServe) Watches() client.Object {
	return &machinelearningv1.SeldonDeployment{}
}

func (r *SeldonModelServe) Apply(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	sd, err := r.convert(md)
	if err != nil {
		return err
	}
	if err := controllerutil.SetOwnerReference(md, sd, r.Client.Scheme()); err != nil {
		return err
	}
	coopy := sd.DeepCopy()
	controllerutil.CreateOrUpdate(ctx, r.Client, sd, func() error {
		sd.Annotations = Mergekvs(coopy.Annotations, sd.Annotations)
		sd.Labels = Mergekvs(coopy.Labels, sd.Labels)
		sd.Spec = coopy.Spec
		return nil
	})

	// nolint: nestif
	if ingresshost := r.IngressHost; ingresshost != "" {
		md.Status.URL = ingresshost + getIngressPath(md)
		if address := sd.Status.Address; address != nil {
			if u, err := url.Parse(address.URL); err == nil {
				md.Status.URL += u.Path
			}
		}
	} else {
		if address := sd.Status.Address; address != nil {
			md.Status.URL = address.URL
		}
	}

	md.Status.RawStatus = ToRawExtension(sd.Status)
	md.Status.Message = sd.Status.Description
	// fill phase
	switch sd.Status.State {
	case machinelearningv1.StatusStateAvailable:
		md.Status.Phase = modelsv1beta1.Running
	case machinelearningv1.StatusStateFailed:
		md.Status.Phase = modelsv1beta1.Failed
	default:
		md.Status.Phase = modelsv1beta1.Pending
	}
	return nil
}

func getIngressPath(md *modelsv1beta1.ModelDeployment) string {
	return "/seldon/" + md.Namespace + "/" + md.Name
}

func (r *SeldonModelServe) convert(md *modelsv1beta1.ModelDeployment) (*machinelearningv1.SeldonDeployment, error) {
	sd := &machinelearningv1.SeldonDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        md.Name,
			Namespace:   md.Namespace,
			Annotations: md.Annotations,
			Labels:      md.Labels,
		},
		Spec: machinelearningv1.SeldonDeploymentSpec{
			Protocol:    machinelearningv1.Protocol(md.Spec.Server.Protocol),
			Annotations: md.Annotations,
			Predictors: []machinelearningv1.PredictorSpec{
				{
					Name:            md.Spec.Server.Name,
					EngineResources: md.Spec.Resources,
					Replicas:        md.Spec.Replicas,
					Graph: machinelearningv1.PredictiveUnit{
						Name:           md.Spec.Server.Name,
						Implementation: implOf(md.Spec.Server.Kind),
						Parameters:     paramsOf(md.Spec.Server.Parameters),
					},
					ComponentSpecs: []*machinelearningv1.SeldonPodSpec{
						{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name:            md.Spec.Server.Name,
										Image:           md.Spec.Server.Image,
										ReadinessProbe:  md.Spec.Server.ReadinessProbe,
										LivenessProbe:   md.Spec.Server.LivenessProbe,
										SecurityContext: md.Spec.Server.SecurityContext,
										Resources:       md.Spec.Resources,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	sd.Spec.Annotations["seldon.io/ingress-path"] = getIngressPath(md) + "/(.*)"
	sd.Spec.Annotations["nginx.ingress.kubernetes.io/rewrite-target"] = "/$1"
	return sd, nil
}

func implOf(name string) *machinelearningv1.PredictiveUnitImplementation {
	impl := machinelearningv1.PredictiveUnitImplementation(name)
	return &impl
}

func paramsOf(params []modelsv1beta1.Parameter) []machinelearningv1.Parameter {
	p := make([]machinelearningv1.Parameter, 0, len(params))
	for _, item := range params {
		p = append(p, machinelearningv1.Parameter{Name: item.Name, Value: item.Value, Type: "STRING"})
	}
	return p
}
