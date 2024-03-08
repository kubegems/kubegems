// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redis

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/pflag"
	"kubegems.io/kubegems/pkg/utils"
)

type Options struct {
	Addr     string `json:"addr,omitempty" description:"redis address"`
	Password string `json:"password,omitempty" description:"redis password"`
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
		Addr:     "", // keep empty to avoid using redis
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
