package options

import (
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/jwt"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/utils/system"
)

type Options struct {
	System   *system.Options   `json:"system,omitempty"`
	Argo     *argo.Options     `json:"argo,omitempty"`
	JWT      *jwt.Options      `json:"jwt,omitempty"`
	LogLevel string            `json:"logLevel,omitempty"`
	Mysql    *database.Options `json:"mysql,omitempty"`
	Redis    *redis.Options    `json:"redis,omitempty"`
}

func DefaultOptions() *Options {
	defaultoptions := &Options{
		Argo:     argo.NewDefaultArgoOptions(),
		JWT:      jwt.DefaultOptions(),
		LogLevel: "debug",
		Mysql:    database.NewDefaultOptions(),
		Redis:    redis.NewDefaultOptions(),
		System:   system.NewDefaultOptions(),
	}
	defaultoptions.System.Listen = ":8020"
	return defaultoptions
}
