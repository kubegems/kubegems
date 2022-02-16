package base

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	msgclient "kubegems.io/pkg/msgbus/client"
	"kubegems.io/pkg/server/define"
	"kubegems.io/pkg/service/aaa"
	"kubegems.io/pkg/service/aaa/audit"
	"kubegems.io/pkg/service/aaa/authorization"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/redis"
)

// BaseHandler is the base handler for all handlers
type BaseHandler struct {
	audit.AuditInterface
	authorization.PermissionChecker
	aaa.UserInterface
	agents     *agents.ClientSet
	database   *database.Database
	redis      *redis.Client
	msgbuscli  msgclient.MessageBusInterface
	cachelayer *models.CacheLayer
}

func NewHandler(server define.ServerInterface) BaseHandler {
	return BaseHandler{
		AuditInterface:    server,
		PermissionChecker: server,
		UserInterface:     server,
		agents:            server.GetAgentsClientSet(),
		msgbuscli:         server.GetMessageBusClient(),
		cachelayer:        server.GetCacheLayer(),
		database:          server.GetDataBase(),
		redis:             server.GetRedis(),
	}
}

func (h *BaseHandler) GetAgents() *agents.ClientSet {
	return h.agents
}

func (h *BaseHandler) GetMessageBusClient() msgclient.MessageBusInterface {
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

func (h *BaseHandler) GetCacheLayer() *models.CacheLayer {
	return h.cachelayer
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
