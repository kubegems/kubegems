package prometheus

import (
	"encoding/json"
	"reflect"
	"testing"

	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func Test_ToAlertmanagerReceiver(t *testing.T) {
	type args struct {
		rec ReceiverConfig
	}
	url1 := "https://baidu.com"
	url2 := "https://google.com"
	tests := []struct {
		name string
		args args
		want v1alpha1.Receiver
	}{
		{
			name: "mult webhook",
			args: args{
				rec: ReceiverConfig{
					Name: "rec-1",
					WebhookConfigs: []WebhookConfig{
						{
							URL: url1,
						},
						{
							URL: url2,
						},
					},
				},
			},
			want: v1alpha1.Receiver{
				Name: "rec-1",
				WebhookConfigs: []v1alpha1.WebhookConfig{
					{
						URL: &url1,
					},
					{
						URL: &url2,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToAlertmanagerReceiver(tt.args.rec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToAlertmanagerReceiver() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isReceiverInUse(t *testing.T) {
	type args struct {
		route    *v1alpha1.Route
		receiver v1alpha1.Receiver
	}
	route1 := v1alpha1.Route{
		GroupInterval: "10s",
		Receiver:      "rec-1",
	}
	route2 := v1alpha1.Route{
		GroupInterval: "10s",
		Receiver:      "rec-2",
	}
	route1Json, _ := json.Marshal(route1)
	route2Json, _ := json.Marshal(route2)
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "recever used",
			args: args{
				route: &v1alpha1.Route{
					Receiver: NullReceiverName,
					Routes: []v1.JSON{
						{
							Raw: route1Json,
						},
						{
							Raw: route2Json,
						},
					},
				},
				receiver: v1alpha1.Receiver{
					Name: "rec-1",
				},
			},
			want: true,
		},
		{
			name: "recever not used",
			args: args{
				route: &v1alpha1.Route{
					Receiver: NullReceiverName,
					Routes: []v1.JSON{
						{
							Raw: route1Json,
						},
						{
							Raw: route2Json,
						},
					},
				},
				receiver: v1alpha1.Receiver{
					Name: "rec-3",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReceiverInUse(tt.args.route, tt.args.receiver); got != tt.want {
				t.Errorf("isReceiverInUse() = %v, want %v", got, tt.want)
			}
		})
	}
}
