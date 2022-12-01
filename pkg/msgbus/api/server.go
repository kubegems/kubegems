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

package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/msgbus/options"
	"kubegems.io/kubegems/pkg/msgbus/switcher"
	"kubegems.io/kubegems/pkg/service/aaa"
	"kubegems.io/kubegems/pkg/service/aaa/auth"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/utils/system"
)

func NewGinServer(opts *options.Options, database *database.Database, redis *redis.Client, ms *switcher.MessageSwitcher) (*gin.Engine, error) {
	r := gin.Default()
	// 初始化需要注册的中间件
	authMiddleware := auth.NewAuthMiddleware(opts.JWT, aaa.NewUserInfoHandler())
	middlewares := []func(*gin.Context){
		authMiddleware.FilterFunc,
	}

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"healthy": "ok"}) })
	for _, md := range middlewares {
		r.Use(md)
	}
	rg := r.Group("/v2")
	msgHandler := &MessageHandler{
		UserInfoHandler: aaa.NewUserInfoHandler(),
		Switcher:        ms,
	}
	msgHandler.RegistRouter(rg)
	return r, nil
}

func RunGinServer(ctx context.Context, options *options.Options, db *database.Database, redis *redis.Client, ms *switcher.MessageSwitcher) error {
	r, err := NewGinServer(options, db, redis, ms)
	if err != nil {
		return err
	}
	return system.ListenAndServeContext(ctx, options.System.Listen, nil, r)
}
