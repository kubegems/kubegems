package agents

import "kubegems.io/pkg/apis/gems"

type Options struct {
	Namespace   string `json:"namespace,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
	ServicePort int    `json:"servicePort,omitempty"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Namespace:   gems.NamespaceSystem,
		ServiceName: "gems-agent",
		ServicePort: 8041,
	}
}
