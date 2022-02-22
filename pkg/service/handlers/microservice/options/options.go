package options

type MicroserviceOptions struct {
	KialiName      string `json:"kialiName,omitempty"`
	KialiNamespace string `json:"kialiNamespace,omitempty"`
}

func NewDefaultOptions() *MicroserviceOptions {
	return &MicroserviceOptions{
		KialiName:      "kiali",
		KialiNamespace: "istio-system",
	}
}
