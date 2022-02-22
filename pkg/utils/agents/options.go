package agents

import "kubegems.io/pkg/apis/gems"

type Options struct {
	Namespace   string `json:"agentNamespace,omitempty" description:"agent service namespace"`
	ServiceName string `json:"agentServiceName,omitempty" description:"agent service name"`
	ServicePort int    `json:"agentServicePort,omitempty" description:"agent service port"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Namespace:   gems.NamespaceSystem,
		ServiceName: "gems-agent",
		ServicePort: 8041,
	}
}
