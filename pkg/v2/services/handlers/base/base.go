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

package base

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
)

// BaseHandler is the base handler for all handlers
type BaseHandler struct {
	agents *agents.ClientSet
	redis  *redis.Client
	db     *database.Database
}

func NewBaseHandler(agents *agents.ClientSet, redis *redis.Client, db *database.Database) BaseHandler {
	return BaseHandler{
		agents: agents,
		redis:  redis,
		db:     db,
	}
}

func (h *BaseHandler) Agents() *agents.ClientSet {
	return h.agents
}

func (h *BaseHandler) Database() *database.Database {
	return h.db
}

func (h *BaseHandler) DB() *gorm.DB {
	return h.db.DB()
}

func (h *BaseHandler) DBWithContext(req *restful.Request) *gorm.DB {
	return h.db.DB().WithContext(req.Request.Context())
}

func (h *BaseHandler) Redis() *redis.Client {
	return h.redis
}

type OnClusterFunc func(ctx context.Context, cli agents.Client) (interface{}, error)

func (h BaseHandler) ClusterFunc(cluster string, fun OnClusterFunc) restful.RouteFunction {
	return func(req *restful.Request, resp *restful.Response) {
		ctx := req.Request.Context()
		cli, err := h.Agents().ClientOf(ctx, cluster)
		if err != nil {
			handlers.BadRequest(resp, err)
			return
		}
		data, err := fun(ctx, cli)
		if err != nil {
			handlers.BadRequest(resp, err)
			return
		}
		if data != nil {
			handlers.BadRequest(resp, err)
		}
		handlers.OK(resp, data)
	}
}

func (h BaseHandler) Execute(ctx context.Context, cluster string, fun func(ctx context.Context, cli agents.Client) error) error {
	cli, err := h.Agents().ClientOf(ctx, cluster)
	if err != nil {
		return err
	}
	return fun(ctx, cli)
}
