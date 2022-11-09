package options

type HubOptions struct {
	Listen string `json:"listen,omitempty"`
	TLS    *TLS   `json:"tls,omitempty"`
}

func NewDefaultHub() *HubOptions {
	return &HubOptions{
		Listen: ":8080",
		TLS:    NewDefaultTLS(),
	}
}
