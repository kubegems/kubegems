package msgbus

import (
	"github.com/spf13/pflag"
	"kubegems.io/pkg/utils"
)

type MsgbusOptions struct {
	Addr string `yaml:"addr"`
}

func (o *MsgbusOptions) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&o.Addr, utils.JoinFlagName(prefix, "addr"), o.Addr, "gems msgbus server addr")
}

func DefaultMsgbusOptions() *MsgbusOptions {
	return &MsgbusOptions{
		Addr: "http://gems-msgbus:8080",
	}
}
