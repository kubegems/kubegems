package options

import (
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/oauth"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/utils/system"
)

type Options struct {
	System   *system.Options   `json:"system,omitempty"`
	Argo     *argo.Options     `json:"argo,omitempty"`
	JWT      *oauth.JWTOptions `json:"jwt,omitempty"`
	LogLevel string            `json:"logLevel,omitempty"`
	Mysql    *database.Options `json:"mysql,omitempty"`
	Redis    *redis.Options    `json:"redis,omitempty"`
}

func DefaultOptions() *Options {
	defaultoptions := &Options{
		Argo:     argo.NewDefaultArgoOptions(),
		JWT:      oauth.NewDefaultJWTOptions(),
		LogLevel: "debug",
		Mysql:    database.NewDefaultOptions(),
		Redis:    redis.NewDefaultOptions(),
		System:   system.NewDefaultOptions(),
	}
	defaultoptions.System.Listen = ":8020"
	return defaultoptions
}
