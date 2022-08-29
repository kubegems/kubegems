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
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	msgclient "kubegems.io/kubegems/pkg/msgbus/client"
	"kubegems.io/kubegems/pkg/service/aaa"
	"kubegems.io/kubegems/pkg/service/aaa/audit"
	"kubegems.io/kubegems/pkg/service/aaa/authorization"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/service/models/cache"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/msgbus"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/utils/set"
)

// BaseHandler is the base handler for all handlers
type BaseHandler struct {
	audit.AuditInterface
	authorization.PermissionManager
	aaa.ContextUserOperator
	agents     *agents.ClientSet
	database   *database.Database
	redis      *redis.Client
	msgbuscli  *msgclient.MsgBusClient
	modelCache *cache.ModelCache
}

func NewHandler(auditi audit.AuditInterface,
	permManager authorization.PermissionManager,
	userif aaa.ContextUserOperator,
	agents *agents.ClientSet,
	database *database.Database,
	redis *redis.Client,
	msgbuscli *msgclient.MsgBusClient,
	modelCache *cache.ModelCache,
) BaseHandler {
	return BaseHandler{
		AuditInterface:      auditi,
		PermissionManager:   permManager,
		ContextUserOperator: userif,
		agents:              agents,
		msgbuscli:           msgbuscli,
		database:            database,
		modelCache:          modelCache,
		redis:               redis,
	}
}

func (h *BaseHandler) GetAgents() *agents.ClientSet {
	return h.agents
}

func (h *BaseHandler) GetMessageBusClient() *msgclient.MsgBusClient {
	return h.msgbuscli
}

func (h *BaseHandler) GetDataBase() *database.Database {
	return h.database
}

func (h *BaseHandler) GetDB() *gorm.DB {
	return h.database.DB()
}

func (h *BaseHandler) GetRedis() *redis.Client {
	return h.redis
}

func (h *BaseHandler) ModelCache() *cache.ModelCache {
	return h.modelCache
}

// OnClusterFunc is the function be called on cluster,the first return value is the http response data,the second is the error
type OnClusterFunc func(ctx context.Context, cli agents.Client) (interface{}, error)

func (h BaseHandler) ClusterFunc(cluster string, fun OnClusterFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		cli, err := h.GetAgents().ClientOf(ctx, cluster)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		data, err := fun(ctx, cli)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		if data != nil {
			handlers.OK(c, data)
		}
	}
}

func (h BaseHandler) Execute(ctx context.Context, cluster string, fun func(ctx context.Context, cli agents.Client) error) error {
	cli, err := h.GetAgents().ClientOf(ctx, cluster)
	if err != nil {
		return err
	}
	return fun(ctx, cli)
}

func (h BaseHandler) SendToMsgbus(c *gin.Context, mutateMsg func(msg *msgclient.MsgRequest)) {
	msg := &msgclient.MsgRequest{
		MessageType:   msgbus.Message,
		Authorization: c.GetHeader("Authorization"),
		ToUsers:       set.NewSet[uint](),
		AffectedUsers: set.NewSet[uint](),
	}
	mutateMsg(msg)

	user, ok := c.Get("current_user")
	if ok {
		msg.Username = user.(*models.User).Username
		msg.Detail = fmt.Sprintf("用户%s%s", msg.Username, msg.Detail)
	}

	go h.GetMessageBusClient().Send(msg)
}
