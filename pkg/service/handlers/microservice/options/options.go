package options

import (
	"github.com/spf13/pflag"
	"kubegems.io/pkg/utils"
)

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

func (o *MicroserviceOptions) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&o.KialiName, utils.JoinFlagName(prefix, "kialiname"), o.KialiName, "kiali service name")
	fs.StringVar(&o.KialiNamespace, utils.JoinFlagName(prefix, "kialinamespace"), o.KialiNamespace, "kiali service namespace")
}
