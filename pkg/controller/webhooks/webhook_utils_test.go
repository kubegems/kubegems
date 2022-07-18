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

package webhooks

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
)

func TestCheckGatewayAndIngressProtocol(t *testing.T) {
	type args struct {
		tg        gemsv1beta1.TenantGateway
		ingresses []networkingv1.Ingress
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "gateway no ConfigMapData",
			args: args{
				tg: gemsv1beta1.TenantGateway{},
				ingresses: []networkingv1.Ingress{
					{
						ObjectMeta: v1.ObjectMeta{
							Annotations: map[string]string{
								"nginx.org/grpc-services": "svc1, svc2",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "gateway not http2",
			args: args{
				tg: gemsv1beta1.TenantGateway{
					Spec: gemsv1beta1.TenantGatewaySpec{
						ConfigMapData: map[string]string{
							"http2": "False",
						},
					},
				},
				ingresses: []networkingv1.Ingress{
					{
						ObjectMeta: v1.ObjectMeta{
							Annotations: map[string]string{
								"nginx.org/grpc-services": "svc1, svc2",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "gateway is http2",
			args: args{
				tg: gemsv1beta1.TenantGateway{
					Spec: gemsv1beta1.TenantGatewaySpec{
						ConfigMapData: map[string]string{
							"http2": "True",
						},
					},
				},
				ingresses: []networkingv1.Ingress{
					{
						ObjectMeta: v1.ObjectMeta{
							Annotations: map[string]string{
								"nginx.org/grpc-services": "svc1, svc2",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "gateway is http2",
			args: args{
				tg: gemsv1beta1.TenantGateway{
					Spec: gemsv1beta1.TenantGatewaySpec{
						ConfigMapData: map[string]string{
							"http2": "true",
						},
					},
				},
				ingresses: []networkingv1.Ingress{
					{
						ObjectMeta: v1.ObjectMeta{
							Annotations: map[string]string{
								"nginx.org/grpc-services": "svc1, svc2",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckGatewayAndIngressProtocol(tt.args.tg, tt.args.ingresses); (err != nil) != tt.wantErr {
				t.Errorf("CheckGatewayAndIngressProtocol() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
