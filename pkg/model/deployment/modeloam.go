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

	oamcommon "github.com/oam-dev/kubevela/apis/core.oam.dev/common"
	oamv1beta1 "github.com/oam-dev/kubevela/apis/core.oam.dev/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"kubegems.io/kubegems/pkg/apis/models"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const OAMModelServeKind = "oam"

type OAMModelServe struct {
	Client client.Client
}

func (r *OAMModelServe) Watches() client.Object {
	return &oamv1beta1.Application{}
}

func (r *OAMModelServe) Apply(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	oamapp, err := r.convert(ctx, md)
	if err != nil {
		return err
	}
	if err := controllerutil.SetOwnerReference(md, oamapp, r.Client.Scheme()); err != nil {
		return err
	}
	if err := ApplyObject(ctx, r.Client, oamapp); err != nil {
		return err
	}
	// fill phase
	md.Status.RawStatus = ToRawExtension(oamapp.Status)
	md.Status.Message = ""
	md.Status.Phase = func() modelsv1beta1.Phase {
		switch oamapp.Status.Phase {
		case oamcommon.ApplicationRunning:
			return modelsv1beta1.Running
		default:
			return modelsv1beta1.Pending
		}
	}()
	return nil
}

// nolint: funlen
func (r *OAMModelServe) convert(ctx context.Context, md *modelsv1beta1.ModelDeployment) (*oamv1beta1.Application, error) {
	const servingPort = 8080
	app := &oamv1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      md.Name,
			Namespace: md.Namespace,
		},
		Spec: oamv1beta1.ApplicationSpec{
			Components: []oamcommon.ApplicationComponent{
				{
					Name: md.Name,
					Type: "webservice",
					Properties: func() *runtime.RawExtension {
						properties := OAMWebServiceProperties{
							Labels:      md.Labels,
							Annotations: md.Annotations,
							Image:       md.Spec.Model.Image,
							ENV: []OAMWebServicePropertiesEnv{
								{
									Name:  "MODEL",
									Value: md.Spec.Model.Name,
								},
							},
							Ports: []OAMWebServicePropertiesPort{
								{Name: "http", Port: servingPort},
							},
						}
						for _, env := range md.Spec.Env {
							properties.ENV = append(properties.ENV, OAMWebServicePropertiesEnv{
								Name:      env.Name,
								Value:     env.Value,
								ValueFrom: env.ValueFrom,
							})
						}
						return ToRawExtension(properties)
					}(),
					Traits: []oamcommon.ApplicationTrait{
						{
							Type: "scaler",
							Properties: models.Properties{
								"replicas": pointer.Int32Deref(md.Spec.Replicas, 1),
							}.ToRawExtension(),
						},
						{
							Type: "json-patch",
							Properties: models.Properties{
								"operations": []any{
									map[string]any{
										"op":    "add",
										"path":  "/spec/template/spec/containers/0/resources",
										"value": md.Spec.Resources,
									},
								},
							}.ToRawExtension(),
						},
						{
							Type: "gateway",
							Properties: models.Properties{
								"domain": md.Spec.Host,
								"http": map[string]interface{}{
									"/": servingPort,
								},
								"classInSpec": true,
							}.ToRawExtension(),
						},
					},
				},
			},
		},
	}
	return app, nil
}
