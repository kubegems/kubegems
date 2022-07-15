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

package workflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-redis/redis/v8"
)

func TestRedisBackend_Sub(t *testing.T) {
	cli := setupRedis(t)

	type fields struct {
		prefix string
		cli    *redis.Client
	}
	type args struct {
		ctx      context.Context
		name     string
		onchange OnChangeFunc
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				prefix: "/test/",
				cli:    cli,
			},
			args: args{
				ctx:  context.Background(),
				name: "test-channel",
				onchange: func(_ context.Context, key string, val []byte) error {
					fmt.Printf("%s->%s\n", key, string(val))
					return nil
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &RedisBackend{
				kvprefix: tt.fields.prefix,
				cli:      tt.fields.cli,
			}
			if err := b.Sub(tt.args.ctx, tt.args.name, tt.args.onchange); (err != nil) != tt.wantErr {
				t.Errorf("RedisBackend.Sub() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
