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
