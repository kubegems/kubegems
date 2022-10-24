package channels

import (
	"net/url"

	"github.com/pkg/errors"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
)

type Webhook struct {
	ChannelType        `json:"channelType"`
	URL                string `json:"url"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
}

func (w *Webhook) ToReceiver(name string) v1alpha1.Receiver {
	cfg := v1alpha1.WebhookConfig{
		URL: &w.URL,
	}
	if w.InsecureSkipVerify {
		cfg.HTTPConfig = &v1alpha1.HTTPConfig{
			TLSConfig: &monv1.SafeTLSConfig{
				InsecureSkipVerify: true,
			},
		}
	}
	return v1alpha1.Receiver{
		Name:           name,
		WebhookConfigs: []v1alpha1.WebhookConfig{cfg},
	}
}

func (w *Webhook) Check() error {
	if _, err := url.ParseRequestURI(w.URL); err != nil {
		return errors.Wrap(err, "url 不合法")
	}
	return nil
}

func (w *Webhook) Test() error {
	return nil
}
