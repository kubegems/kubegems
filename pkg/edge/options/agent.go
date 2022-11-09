package options

type AgentOptions struct {
	TLS         *TLS   `json:"tls,omitempty"`
	EdgeHubAddr string `json:"edgeHubAddr,omitempty"`
}

func NewDefaultAgentOptions() *AgentOptions {
	return &AgentOptions{
		TLS:         NewDefaultTLS(),
		EdgeHubAddr: "127.0.0.1:8080",
	}
}
