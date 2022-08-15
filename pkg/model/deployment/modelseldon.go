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
	"strconv"
	"strings"

	machinelearningv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const SeldonModelServeKind = "seldon"

type SeldonModelServe struct {
	Client        client.Client
	IngressHost   string
	IngressScheme string
}

func (r *SeldonModelServe) Watches() client.Object {
	return &machinelearningv1.SeldonDeployment{}
}

func (r *SeldonModelServe) Apply(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	sd, err := r.convert(ctx, md)
	if err != nil {
		return err
	}
	if err := controllerutil.SetOwnerReference(md, sd, r.Client.Scheme()); err != nil {
		return err
	}
	coopy := sd.DeepCopy()
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, sd, func() error {
		sd.Annotations = Mergekvs(coopy.Annotations, sd.Annotations)
		sd.Labels = Mergekvs(coopy.Labels, sd.Labels)
		sd.Spec = coopy.Spec
		return nil
	})
	if err != nil {
		return err
	}

	if err := r.completeStatusURL(ctx, md, sd); err != nil {
		return err
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

func (r *SeldonModelServe) completeStatusURL(ctx context.Context, md *modelsv1beta1.ModelDeployment, sd *machinelearningv1.SeldonDeployment) error {
	// find same name ingress
	ingress := &networkingv1.Ingress{}
	// ignore error

	u := &url.URL{
		Scheme: r.IngressScheme,
		Host:   r.IngressHost,
		Path:   getIngressPath(md),
	}

	_ = r.Client.Get(ctx, client.ObjectKey{Name: md.Name, Namespace: md.Namespace}, ingress)
	for _, rule := range ingress.Spec.Rules {
		if host := rule.Host; host != "" {
			u.Host = host
			break
		}
	}
	if gatewayName := md.Spec.Ingress.GatewayName; gatewayName != "" {
		gateway := &gemsv1beta1.TenantGateway{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: gatewayName}, gateway); err != nil {
			return err
		}
		for _, gatewayport := range gateway.Status.Ports {
			if gatewayport.Name == u.Scheme {
				u.Host += ":" + strconv.Itoa(int(gatewayport.NodePort))
				break
			}
		}
	}
	if address := sd.Status.Address; address != nil {
		if sdurl, err := url.Parse(address.URL); err == nil {
			u.Path += sdurl.Path
		}
	}

	md.Status.URL = u.String()
	return nil
}

func getIngressPath(md *modelsv1beta1.ModelDeployment) string {
	return "/seldon/" + md.Namespace + "/" + md.Name
}

func (r *SeldonModelServe) convert(ctx context.Context, md *modelsv1beta1.ModelDeployment) (*machinelearningv1.SeldonDeployment, error) {
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
					EngineResources: md.Spec.Server.Resources,
					Replicas:        md.Spec.Replicas,
					Graph: machinelearningv1.PredictiveUnit{
						Name:                    md.Spec.Server.Name,
						Implementation:          implOf(md.Spec.Server.Kind),
						Parameters:              paramsOf(md.Spec.Server.Parameters),
						ModelURI:                md.Spec.Model.URL,
						StorageInitializerImage: md.Spec.Server.StorageInitializerImage,
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
										Resources:       md.Spec.Server.Resources,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ingressclass, err := r.getIngressClass(ctx, md)
	if err != nil {
		return nil, err
	}
	sd.Spec.Annotations["seldon.io/ingress-class-name"] = ingressclass
	sd.Spec.Annotations["seldon.io/ingress-host"] = md.Spec.Ingress.Host
	sd.Spec.Annotations["seldon.io/ingress-path"] = getIngressPath(md) + "/(.*)"
	sd.Spec.Annotations["nginx.ingress.kubernetes.io/rewrite-target"] = "/$1"
	return sd, nil
}

func (r *SeldonModelServe) getIngressClass(ctx context.Context, md *modelsv1beta1.ModelDeployment) (string, error) {
	if md.Spec.Ingress.ClassName != "" {
		return md.Spec.Ingress.ClassName, nil
	}
	if getewayname := md.Spec.Ingress.GatewayName; getewayname != "" {
		gateway := &gemsv1beta1.TenantGateway{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: getewayname}, gateway); err != nil {
			return "", err
		}
		return gateway.Spec.IngressClass, nil
	}
	return "", nil
}

func implOf(name string) *machinelearningv1.PredictiveUnitImplementation {
	implName := strings.ToUpper(strings.Replace(name, "-", "_", -1))
	impl := machinelearningv1.PredictiveUnitImplementation(implName)
	return &impl
}

func paramsOf(params []modelsv1beta1.Parameter) []machinelearningv1.Parameter {
	p := make([]machinelearningv1.Parameter, 0, len(params))
	for _, item := range params {
		p = append(p, machinelearningv1.Parameter{Name: item.Name, Value: item.Value, Type: "STRING"})
	}
	return p
}
