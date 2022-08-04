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

	machinelearningv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const SeldonModelServeKind = "seldon"

type SeldonModelServe struct {
	Client client.Client
}

func (r *SeldonModelServe) Watches() client.Object {
	return &machinelearningv1.SeldonDeployment{}
}

func (r *SeldonModelServe) Apply(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	sd, objects, err := r.convert(md)
	if err != nil {
		return err
	}
	if err := controllerutil.SetOwnerReference(md, sd, r.Client.Scheme()); err != nil {
		return err
	}
	if err := ApplyObject(ctx, r.Client, sd); err != nil {
		return err
	}
	for _, object := range objects {
		if err := controllerutil.SetOwnerReference(md, object, r.Client.Scheme()); err != nil {
			return err
		}
		if err := ApplyObject(ctx, r.Client, object); err != nil {
			return err
		}
	}

	// TODO: add a status update here
	md.Status.URL = sd.Status.Address.URL

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

func (r *SeldonModelServe) convert(md *modelsv1beta1.ModelDeployment) (*machinelearningv1.SeldonDeployment, []client.Object, error) {
	sd := &machinelearningv1.SeldonDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        md.Name,
			Namespace:   md.Namespace,
			Annotations: md.Annotations,
			Labels:      md.Labels,
		},
		Spec: machinelearningv1.SeldonDeploymentSpec{
			Predictors: []machinelearningv1.PredictorSpec{
				updateDefaultPredicator(machinelearningv1.PredictorSpec{}, md),
			},
		},
	}
	return sd, []client.Object{}, nil
}

func updateDefaultPredicator(predictor machinelearningv1.PredictorSpec, md *modelsv1beta1.ModelDeployment) machinelearningv1.PredictorSpec {
	predictor.Name = "default"
	predictor.Graph.Name = "model"
	if predictor.Graph.Endpoint == nil {
		predictor.Graph.Endpoint = getEndpoint(md)
	}
	if predictor.Graph.Implementation == nil {
		predictor.Graph.Implementation = getServerImpl(md.Spec.Model.ServerType)
	}
	if predictor.Graph.ModelURI == "" {
		predictor.Graph.ModelURI = md.Spec.Model.URL
	}
	// append params to predictor
	for _, param := range md.Spec.Model.Prameters {
		predictor.Graph.Parameters = append(predictor.Graph.Parameters, machinelearningv1.Parameter{
			Name:  param.Name,
			Value: param.Value,
			Type:  "STRING",
		})
	}
	// override default server image

	var maincontainer *corev1.Container
	for _, item := range predictor.ComponentSpecs {
		for j, container := range item.Spec.Containers {
			if container.Name != predictor.Graph.Name {
				continue
			}
			maincontainer = &item.Spec.Containers[j]
		}
	}
	if maincontainer == nil {
		spec := &machinelearningv1.SeldonPodSpec{
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: predictor.Graph.Name}}},
		}
		maincontainer = &spec.Spec.Containers[0]
		predictor.ComponentSpecs = append(predictor.ComponentSpecs, spec)
	}

	// override image
	if imageOverride := md.Spec.Model.Image; imageOverride != "" {
		maincontainer.Image = imageOverride
	}
	// override probes
	if containeroverride := md.Spec.Model.ContainerSpecOverrride; containeroverride != nil {
		maincontainer.ReadinessProbe = containeroverride.ReadinessProbe
		maincontainer.LivenessProbe = containeroverride.LivenessProbe
		maincontainer.StartupProbe = containeroverride.StartupProbe
	}
	return predictor
}

func getServerImpl(name string) *machinelearningv1.PredictiveUnitImplementation {
	if name == "" {
		return nil
	}
	impl := machinelearningv1.PredictiveUnitImplementation(name)
	return &impl
}

func getEndpoint(md *modelsv1beta1.ModelDeployment) *machinelearningv1.Endpoint {
	httpport, grpcport := int32(0), int32(0)
	for _, port := range md.Spec.Ports {
		switch port.Name {
		case "grpc":
			grpcport = port.ContainerPort
		case "http":
			httpport = port.ContainerPort
		}
	}
	return &machinelearningv1.Endpoint{GrpcPort: grpcport, HttpPort: httpport}
}
