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

package mongo

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Options struct {
	Addr     string `json:"addr,omitempty" description:"mongodb address"`
	Database string `json:"database,omitempty" description:"mongodb database"`
	Username string `json:"username,omitempty" description:"mongodb username"`
	Password string `json:"password,omitempty" description:"mongodb password"`
}

func DefaultOptions() *Options {
	return &Options{
		Addr:     "mongo:27017",
		Database: "models",
		Username: "",
		Password: "",
	}
}

func New(ctx context.Context, opt *Options) (*mongo.Client, *mongo.Database, error) {
	mongocli, db, err := NewLazy(ctx, opt)
	if err != nil {
		return nil, nil, err
	}
	if err := mongocli.Connect(ctx); err != nil {
		return nil, nil, err
	}
	if err := mongocli.Ping(ctx, nil); err != nil {
		return nil, nil, err
	}
	return mongocli, db, nil
}

func NewLazy(ctx context.Context, opt *Options) (*mongo.Client, *mongo.Database, error) {
	mongoopt := &options.ClientOptions{
		Hosts: strings.Split(opt.Addr, ","),
	}
	if opt.Username != "" && opt.Password != "" {
		mongoopt.Auth = &options.Credential{
			Username: opt.Username,
			Password: opt.Password,
		}
	}
	mongocli, err := mongo.NewClient(mongoopt)
	if err != nil {
		return nil, nil, err
	}
	return mongocli, mongocli.Database(opt.Database), nil
}
