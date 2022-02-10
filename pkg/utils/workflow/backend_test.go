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
