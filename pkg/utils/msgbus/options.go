package msgbus

import (
	"github.com/kubegems/gems/pkg/utils"
	"github.com/spf13/pflag"
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
