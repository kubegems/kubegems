package system

import (
	"time"

	"kubegems.io/pkg/apis/gems"
)

type Options struct {
	Listen           string        `json:"listen,omitempty" description:"listen address"`
	JwtExpire        time.Duration `json:"jwtExpire,omitempty" description:"jwt expire time"`
	JWTCert          string        `json:"jwtCert,omitempty" description:"jwt cert file"`
	JWTKey           string        `json:"jwtKey,omitempty" description:"jwt key file"`
	AgentNamespace   string        `json:"agentNamespace,omitempty" description:"agent namespace"`
	AgentServiceName string        `json:"agentServiceName,omitempty" description:"agent service name"`
	AgentServicePort int           `json:"agentServicePort,omitempty" description:"agent service port"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Listen:           ":8020",
		JwtExpire:        24 * time.Hour, // 24小时
		JWTCert:          "certs/jwt/tls.crt",
		JWTKey:           "certs/jwt/tls.key",
		AgentNamespace:   gems.NamespaceSystem,
		AgentServiceName: "gems-agent",
		AgentServicePort: 8041,
	}
}
