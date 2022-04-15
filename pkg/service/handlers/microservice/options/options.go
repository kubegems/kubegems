package options

import "kubegems.io/pkg/apis/gems"

type MicroserviceOptions struct {
	KialiName         string `json:"kialiName,omitempty"`
	KialiNamespace    string `json:"kialiNamespace,omitempty"`
	GatewayNamespace  string `json:"gatewayNamespace,omitempty"`
	IstioNamespace    string `json:"istioNamespace,omitempty"`
	IstioOperatorName string `json:"istioOperatorName,omitempty"`
}

func NewDefaultOptions() *MicroserviceOptions {
	return &MicroserviceOptions{
		KialiName:         "kiali",
		KialiNamespace:    "istio-system",
		GatewayNamespace:  gems.NamespaceGateway,
		IstioNamespace:    "istio-system",
		IstioOperatorName: "gems-istio",
	}
}
