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
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// Backend 作为后端的数据存储，需要一致性支持
// 需要支持队列和kv存储
// 队列用于分发任务，kv存储用于存储状态等持久化数据。

const (
	DefaultGroup = "workflow-group"
)

type OnChangeFunc func(ctx context.Context, key string, val []byte) error

type Backend interface {
	// 队列
	Sub(ctx context.Context, name string, onchange OnChangeFunc, opts ...SubOption) error
	// 这里的sub要求多个消费者共享同一个topic下的数据，且无重复。
	Pub(ctx context.Context, name string, key string, val []byte) error

	// kv存储
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, val []byte, ttl ...time.Duration) error
	Del(ctx context.Context, key string) error
	List(ctx context.Context, keyprefix string) (map[string][]byte, error)
	Watch(ctx context.Context, key string, onchange OnChangeFunc) error
}

type RedisBackend struct {
	kvprefix    string
	steamprefix string
	cli         *redis.Client
}

func NewRedisBackend(addr, username, password string) *RedisBackend {
	cli := redis.NewClient(&redis.Options{Addr: addr, Username: username, Password: password})
	return NewRedisBackendFromClient(cli)
}

func NewRedisBackendFromClient(c *redis.Client) *RedisBackend {
	return &RedisBackend{
		kvprefix:    "/workflow-store/",
		steamprefix: "/workflow-queue/",
		cli:         c,
	}
}

type SubOptions struct {
	AutoACK     bool // 自动确认，无论结果是否为 error
	Concurrency int  // 支持的并发数量
}

type SubOption func(o *SubOptions)

func WithConcurrency(con int) SubOption {
	return func(o *SubOptions) { o.Concurrency = con }
}

func WithAutoACK(ack bool) SubOption {
	return func(o *SubOptions) { o.AutoACK = ack }
}

// 队列
func (b *RedisBackend) Sub(ctx context.Context, name string, onchange OnChangeFunc, opts ...SubOption) error {
	options := &SubOptions{Concurrency: 1}
	for _, opt := range opts {
		opt(options)
	}

	keyprefix := b.steamprefix + name

	consumergroup := DefaultGroup
	// 创建消费组
	// https://redis.io/commands/xgroup-create
	if err := b.cli.XGroupCreateMkStream(ctx, keyprefix, consumergroup, "0").Err(); err != nil {
		// 这里只能先用 str 来判断
		if !strings.Contains(err.Error(), "exists") {
			return err
		}
	}

	// concurrent
	concurrentchan := make(chan struct{}, options.Concurrency)

	shouldconsumeunacked := true
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// 消费
			// 在第一次启动时，消费上次未确认的任务
			// 后续仅消费新任务，可保证正在消费的任务不会重新拿到
			// 依次执行:
			// 1.  xreadgroup ... 0-0 // 消费上次中断未消费任务
			// 2.  xreadgroup ... >	  // 阻塞直到新消息
			ids := ">"
			if shouldconsumeunacked {
				shouldconsumeunacked = false
				ids = "0"
			}
			// https://redis.io/commands/XREADGROUP
			result, err := b.cli.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group: consumergroup,
				Consumer: func() string {
					hostname, _ := os.Hostname()
					return hostname
				}(),
				Streams: []string{
					keyprefix, ids,
				},
				Block: 0,
			}).Result()
			if err != nil {
				return err
			}

			for _, msgs := range result {
				for _, msg := range msgs.Messages {
					for k, v := range msg.Values {
						val := []byte{}
						switch data := v.(type) {
						case string:
							val = []byte(data)
						case []byte:
							val = data
						}

						select {
						case <-ctx.Done():
							return nil
						case concurrentchan <- struct{}{}:
							go func(k string, v []byte) {
								if err := onchange(ctx, k, v); err != nil {
									if options.AutoACK {
										// ack
										b.cli.XAck(ctx, msgs.Stream, consumergroup, msg.ID)
									}
								} else {
									// ack
									b.cli.XAck(ctx, msgs.Stream, consumergroup, msg.ID)
								}

								// put it back
								<-concurrentchan
							}(k, val)

						}
					}
				}
			}
		}
	}
}

func (b *RedisBackend) Pub(ctx context.Context, name string, key string, val []byte) error {
	keyprefix := b.steamprefix + name
	return b.cli.XAdd(ctx, &redis.XAddArgs{
		Stream: keyprefix,
		Values: map[string]interface{}{key: val},
	}).Err()
}

// kv存储
func (b *RedisBackend) Put(ctx context.Context, key string, val []byte, ttl ...time.Duration) error {
	prefixedKey := b.kvprefix + key
	set := b.cli.Set(ctx, prefixedKey, val, 0)
	return set.Err()
}

func (b *RedisBackend) Del(ctx context.Context, key string) error {
	prefixedKey := b.kvprefix + key
	return b.cli.Del(ctx, prefixedKey).Err()
}

func (b *RedisBackend) Get(ctx context.Context, key string) ([]byte, error) {
	prefixedKey := b.kvprefix + key
	get := b.cli.Get(ctx, prefixedKey)
	return get.Bytes()
}

func (b *RedisBackend) List(ctx context.Context, keyprefix string) (map[string][]byte, error) {
	prefixedKey := b.kvprefix + keyprefix
	iter := b.cli.Scan(ctx, 0, prefixedKey+"*", 0).Iterator()

	list := map[string][]byte{}
	for iter.Next(ctx) {

		if err := iter.Err(); err != nil {
			return nil, err
		}

		key := iter.Val()
		val, err := b.cli.Get(ctx, key).Bytes()
		if err != nil {
			return nil, err
		}

		itemkey := strings.TrimPrefix(key, prefixedKey)
		list[itemkey] = val
	}
	return list, nil
}

func (b *RedisBackend) Watch(ctx context.Context, key string, onchange OnChangeFunc) error {
	// https://redis.io/topics/notifications
	_ = b.cli.ConfigSet(ctx, "notify-keyspace-events", "KA")

	channelprefix := fmt.Sprintf("__keyspace@%d__:%s", b.cli.Options().DB, b.kvprefix)
	pubsub := b.cli.PSubscribe(ctx, channelprefix+key+"*")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				return err
			}
			name := strings.TrimPrefix(msg.Channel, channelprefix)
			val, err := b.Get(ctx, name)
			if err != nil {
				continue
			}
			if err := onchange(ctx, name, val); err != nil {
				return err
			}
		}
	}
}
