package system

import (
	"time"

	"github.com/spf13/pflag"
	"kubegems.io/pkg/utils"
)

type SystemOptions struct {
	Listen           string        `yaml:"listen"`
	JwtExpire        time.Duration `yaml:"jwtexpire"`
	EnableAudit      bool          `yaml:"enableaudit"`
	JWTCert          string        `yaml:"jwtcert"`
	JWTKey           string        `yaml:"jwtkey"`
	AgentTimeout     int           `yaml:"agentTimeout"`
	AgentNamespace   string        `yaml:"agentNamespace"`
	AgentServiceName string        `yaml:"agentServiceName"`
	AgentServicePort int           `yaml:"agentServicePort"`
}

func (s *SystemOptions) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&s.Listen, utils.JoinFlagName(prefix, "listen"), ":8020", "system listen addr")
	fs.DurationVar(&s.JwtExpire, utils.JoinFlagName(prefix, "jwtexpire"), time.Hour*24, "jwt token expire time duration")
	fs.BoolVar(&s.EnableAudit, utils.JoinFlagName(prefix, "enableaudit"), true, "enable audit")
	fs.IntVar(&s.AgentTimeout, utils.JoinFlagName(prefix, "agentTimeout"), 30, "agent request timeout seconds, default 30")
	fs.StringVar(&s.AgentNamespace, utils.JoinFlagName(prefix, "agentNamespace"), s.AgentNamespace, "agent service namespace")
	fs.StringVar(&s.AgentServiceName, utils.JoinFlagName(prefix, "agentServiceName"), s.AgentServiceName, "agent service name")
	fs.IntVar(&s.AgentServicePort, utils.JoinFlagName(prefix, "agentServicePort"), s.AgentServicePort, "agent service port")
}

func NewDefaultOptions() *SystemOptions {
	return &SystemOptions{
		Listen:           ":8020",
		JwtExpire:        24 * time.Hour, // 24小时
		EnableAudit:      true,
		JWTCert:          "certs/jwt/tls.crt",
		JWTKey:           "certs/jwt/tls.key",
		AgentTimeout:     10,
		AgentNamespace:   "gemcloud-system",
		AgentServiceName: "gems-agent",
		AgentServicePort: 8041,
	}
}
