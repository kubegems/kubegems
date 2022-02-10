package prometheus

import (
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
)

var (
	DefaultReceiverName = "gemcloud-default-webhook"
	DefaultReceiverURL  = "http://gems-agent.gemcloud-system:8041/alert"
	DefaultReceiver     = v1alpha1.Receiver{
		Name: DefaultReceiverName,
		WebhookConfigs: []v1alpha1.WebhookConfig{
			{
				URL: &DefaultReceiverURL,
			},
		},
	}

	NullReceiverName = "null"
	NullReceiver     = v1alpha1.Receiver{Name: NullReceiverName}
)
