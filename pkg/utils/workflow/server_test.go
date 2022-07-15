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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func setupRedis(t *testing.T) *redis.Client {
	s := miniredis.RunT(t)
	addr := s.Addr()

	// addr = "127.0.0.1:6379"
	return redis.NewClient(&redis.Options{Addr: addr})
}

type DemoArgs struct {
	Foo string `json:"foo,omitempty"`
}

var registeredfunc = map[string]interface{}{
	"echo": func(_ context.Context, val string) error {
		fmt.Print(val)
		return nil
	},
	"clean": func(val string) error {
		fmt.Printf("clean %s", val)
		return nil
	},
	"inobj": func(_ context.Context, arg DemoArgs) error {
		fmt.Printf("inobj called:arg=%v", arg)
		return nil
	},
	"inobjpointer": func(_ context.Context, arg *DemoArgs) error {
		fmt.Printf("inobjpointer called:arg=%v", arg)
		return nil
	},
	"now": func() string {
		return time.Now().String()
	},
	"variadic": func(a DemoArgs, s ...string) error {
		fmt.Printf("isVariadic called: arg=%v,variadic=%v", a, s)
		return nil
	},
	"arr": func(strs []string) {
		fmt.Printf("arr called:arg=%v", strs)
	},
}

func TestServer_Run(t *testing.T) {
	rediscli := setupRedis(t)

	type fields struct {
		backend    Backend
		registered map[string]interface{}
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args

		task    Task
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				backend:    NewRedisBackendFromClient(rediscli),
				registered: registeredfunc,
			},
			args: args{
				ctx: context.Background(),
			},
			task: Task{
				Name: "all",
				Steps: []Step{
					{
						Name:     "prepare",
						Function: "echo",
						Args:     []interface{}{"hello world"},
					},
					{
						Name:     "what-time",
						Function: "now",
					},
					{
						Name:     "finished",
						Function: "clean",
						Args:     []interface{}{"up"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				backend:    tt.fields.backend,
				registered: tt.fields.registered,
			}

			if err := NewClientFromBackend(s.backend).SubmitTask(tt.args.ctx, tt.task); err != nil {
				t.Error(err)
				return
			}

			if err := s.Run(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Server.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServer_execute(t *testing.T) {
	type fields struct {
		backend    Backend
		registered map[string]interface{}
	}
	type args struct {
		ctx  context.Context
		task *jsonArgsStep
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
				registered: registeredfunc,
			},
			args: args{
				ctx: context.Background(),
				task: &jsonArgsStep{
					Function: "inobj",
					Args: []json.RawMessage{
						json.RawMessage(`{"foo":"bar"}`),
					},
				},
			},
		},
		{
			name: "",
			fields: fields{
				registered: registeredfunc,
			},
			args: args{
				ctx: context.Background(),
				task: &jsonArgsStep{
					Function: "inobjpointer",
					Args: []json.RawMessage{
						json.RawMessage(`{"foo":"bar"}`),
					},
				},
			},
		},
		{
			name: "variadic",
			fields: fields{
				registered: registeredfunc,
			},
			args: args{
				ctx: context.Background(),
				task: &jsonArgsStep{
					Function: "variadic",
					Args: []json.RawMessage{
						json.RawMessage(`{"foo":"bar"}`),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Server{
				backend:    tt.fields.backend,
				registered: tt.fields.registered,
			}
			if err := n.execute(tt.args.ctx, tt.args.task); (err != nil) != tt.wantErr {
				t.Errorf("Server.execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
