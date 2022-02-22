package agents

import "kubegems.io/pkg/apis/gems"

type Options struct {
	AgentNamespace   string `json:"agentNamespace,omitempty" description:"agent namespace"`
	AgentServiceName string `json:"agentServiceName,omitempty" description:"agent service name"`
	AgentServicePort int    `json:"agentServicePort,omitempty" description:"agent service port"`
}

func NewDefaultOptions() *Options {
	return &Options{
		AgentNamespace:   gems.NamespaceSystem,
		AgentServiceName: "gems-agent",
		AgentServicePort: 8041,
	}
}
