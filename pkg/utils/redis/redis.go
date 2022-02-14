package redis

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/kubegems/gems/pkg/utils"
	"github.com/spf13/pflag"
)

type Options struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
}

func (o *Options) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&o.Addr, utils.JoinFlagName(prefix, "addr"), o.Addr, "redis address")
	fs.StringVar(&o.Password, utils.JoinFlagName(prefix, "password"), o.Password, "redis password")
}

func (o *Options) ToDsn(db int) string {
	if len(o.Password) == 0 {
		return fmt.Sprintf("redis://%s/%v", o.Addr, db)
	} else {
		return fmt.Sprintf("redis://%s@%s/%v", o.Password, o.Addr, db)
	}
}

func NewDefaultOptions() *Options {
	return &Options{
		Addr:     "gems-redis:6379",
		Password: "",
	}
}

type Client struct {
	*redis.Client
}

func NewClient(options *Options) (*Client, error) {
	cli := redis.NewClient(&redis.Options{
		Addr:     options.Addr,
		Password: options.Password,
	})

	return &Client{Client: cli}, nil
}
