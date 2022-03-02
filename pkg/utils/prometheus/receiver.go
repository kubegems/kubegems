package prometheus

import (
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
)

var (
	DefaultReceiverName = "gemcloud-default-webhook"
	DefaultReceiverURL  = "https://gems-agent.gemcloud-system:8041/alert"
	DefaultReceiver     = v1alpha1.Receiver{
		Name: DefaultReceiverName,
		WebhookConfigs: []v1alpha1.WebhookConfig{
			{
				URL: &DefaultReceiverURL,
				HTTPConfig: &v1alpha1.HTTPConfig{
					TLSConfig: &v1.SafeTLSConfig{
						InsecureSkipVerify: true,
					},
				},
			},
		},
	}

	NullReceiverName = "null"
	NullReceiver     = v1alpha1.Receiver{Name: NullReceiverName}
)
